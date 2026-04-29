package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/proto"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// WorkerClient NodeWorker gRPC 客户端，供 Engine 调用 NodeWorker 服务
// 使用地址池，按 Worker 地址动态管理连接
type WorkerClient struct {
	mu    sync.Mutex
	conns map[string]*workerConn
}

type workerConn struct {
	conn   *grpc.ClientConn
	client proto.NodeWorkerServiceClient
}

// NewWorkerClient 创建 WorkerClient（懒加载连接）
func NewWorkerClient() *WorkerClient {
	return &WorkerClient{
		conns: make(map[string]*workerConn),
	}
}

// getConn 根据地址获取或新建 gRPC 连接
func (wc *WorkerClient) getConn(address string) (*workerConn, error) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if c, ok := wc.conns[address]; ok {
		return c, nil
	}

	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to worker at %s: %w", address, err)
	}
	wc.conns[address] = &workerConn{
		conn:   conn,
		client: proto.NewNodeWorkerServiceClient(conn),
	}
	return wc.conns[address], nil
}

// CloseAll 关闭所有连接，通常在 Engine 关闭时调用
func (wc *WorkerClient) CloseAll() {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	for addr, c := range wc.conns {
		if err := c.conn.Close(); err != nil {
			log.Warn().Str("address", addr).Err(err).Msg("close worker connection failed")
		}
	}
	wc.conns = make(map[string]*workerConn)
}

// CloseWorker 关闭指定地址的连接（Worker 下线时调用）
func (wc *WorkerClient) CloseWorker(address string) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if c, ok := wc.conns[address]; ok {
		_ = c.conn.Close()
		delete(wc.conns, address)
	}
}

// AssignTask 向指定 Worker 分配任务
func (wc *WorkerClient) AssignTask(address string, task *types.Task) error {
	c, err := wc.getConn(address)
	if err != nil {
		return err
	}

	configBytes, err := json.Marshal(task.Config)
	if err != nil {
		return fmt.Errorf("marshal task config error: %w", err)
	}
	inputBytes, err := json.Marshal(task.InputData)
	if err != nil {
		return fmt.Errorf("marshal task input_data error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	start := time.Now()
	resp, err := c.client.AssignTask(ctx, &proto.AssignTaskRequest{
		TaskId:             task.TaskID,
		WorkflowInstanceId: task.InstanceID,
		WorkflowNodeId:     task.NodeID,
		NodeType:           task.NodeType,
		Configuration:      configBytes,
		InputData:          inputBytes,
		Timeout:            int32(task.Timeout),
	})
	elapsed := time.Since(start)

	if err != nil {
		log.Error().Err(err).
			Str("worker_addr", address).
			Str("instance_id", task.InstanceID).
			Str("node_id", task.NodeID).
			Dur("elapsed", elapsed).
			Msg("AssignTask rpc failed")
		return fmt.Errorf("assign task rpc error: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("assign task failed: code=%d msg=%s", resp.Code, resp.Message)
	}
	log.Info().
		Str("worker_addr", address).
		Str("task_id", task.TaskID).
		Str("instance_id", task.InstanceID).
		Str("node_id", task.NodeID).
		Dur("elapsed", elapsed).
		Msg("AssignTask success")
	return nil
}

// CancelTask 取消指定 Worker 上的任务
func (wc *WorkerClient) CancelTask(address, instanceID, nodeID, reason string) error {
	c, err := wc.getConn(address)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	start := time.Now()
	resp, err := c.client.CancelTask(ctx, &proto.CancelTaskRequest{
		WorkflowInstanceId: instanceID,
		WorkflowNodeId:     nodeID,
		Reason:             reason,
	})
	elapsed := time.Since(start)

	if err != nil {
		log.Error().Err(err).
			Str("worker_addr", address).
			Str("instance_id", instanceID).
			Str("node_id", nodeID).
			Dur("elapsed", elapsed).
			Msg("CancelTask rpc failed")
		return fmt.Errorf("cancel task rpc error: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("cancel task failed: code=%d msg=%s", resp.Code, resp.Message)
	}
	log.Info().
		Str("worker_addr", address).
		Str("instance_id", instanceID).
		Str("node_id", nodeID).
		Dur("elapsed", elapsed).
		Msg("CancelTask success")
	return nil
}

// PauseTask 暂停指定 Worker 上正在执行的任务
func (wc *WorkerClient) PauseTask(address, instanceID, nodeID string) error {
	c, err := wc.getConn(address)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	resp, err := c.client.PauseTask(ctx, &proto.PauseTaskRequest{
		WorkflowInstanceId: instanceID,
		WorkflowNodeId:     nodeID,
	})
	if err != nil {
		return fmt.Errorf("pause task rpc error: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("pause task failed: code=%d msg=%s", resp.Code, resp.Message)
	}
	return nil
}

// HealthCheck 对指定 Worker 发起健康检查，返回 true 表示 Worker 正常
func (wc *WorkerClient) HealthCheck(address, workerID string) bool {
	c, err := wc.getConn(address)
	if err != nil {
		log.Warn().Str("address", address).Err(err).Msg("HealthCheck: connect failed")
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.HealthCheck(ctx, &proto.HealthCheckRequest{
		WorkerId: workerID,
	})
	if err != nil {
		log.Warn().Str("worker_id", workerID).Err(err).Msg("HealthCheck rpc failed")
		return false
	}
	return resp.Success
}



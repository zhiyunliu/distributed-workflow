package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/proto"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
)

const (
	defaultTimeout   = 5 * time.Second
	heartbeatTimeout = 3 * time.Second
)

// EngineClient Engine gRPC 客户端，供 NodeWorker 调用 Engine 服务
type EngineClient struct {
	conn   *grpc.ClientConn
	client proto.EngineServiceClient
	addr   string
}

// NewEngineClient 创建 Engine gRPC 客户端，建立与 Engine 的连接
func NewEngineClient(addr string) (*EngineClient, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to engine at %s: %w", addr, err)
	}
	return &EngineClient{
		conn:   conn,
		client: proto.NewEngineServiceClient(conn),
		addr:   addr,
	}, nil
}

// Close 关闭 gRPC 连接
func (c *EngineClient) Close() error {
	return c.conn.Close()
}

// RegisterWorker 注册 Worker 节点
func (c *EngineClient) RegisterWorker(workerID, address, workerIP string, capabilities map[string]int) error {
	caps := make(map[string]int32, len(capabilities))
	for k, v := range capabilities {
		caps[k] = int32(v)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	start := time.Now()
	resp, err := c.client.RegisterWorker(ctx, &proto.RegisterWorkerRequest{
		WorkerId:     workerID,
		Address:      address,
		WorkerIp:     workerIP,
		Capabilities: caps,
	})
	elapsed := time.Since(start)

	if err != nil {
		log.Error().Err(err).Str("addr", c.addr).Dur("elapsed", elapsed).Msg("RegisterWorker rpc failed")
		return fmt.Errorf("register worker rpc error: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("register worker failed: code=%d msg=%s", resp.Code, resp.Message)
	}
	log.Info().Str("worker_id", workerID).Dur("elapsed", elapsed).Msg("RegisterWorker success")
	return nil
}

// UnregisterWorker 注销 Worker 节点
func (c *EngineClient) UnregisterWorker(workerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	start := time.Now()
	resp, err := c.client.UnregisterWorker(ctx, &proto.UnregisterWorkerRequest{
		WorkerId: workerID,
	})
	elapsed := time.Since(start)

	if err != nil {
		log.Error().Err(err).Str("worker_id", workerID).Dur("elapsed", elapsed).Msg("UnregisterWorker rpc failed")
		return fmt.Errorf("unregister worker rpc error: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("unregister worker failed: code=%d msg=%s", resp.Code, resp.Message)
	}
	log.Info().Str("worker_id", workerID).Dur("elapsed", elapsed).Msg("UnregisterWorker success")
	return nil
}

// Heartbeat 发送心跳，上报当前负载
func (c *EngineClient) Heartbeat(workerID string, currentLoad map[string]int) error {
	load := make(map[string]int32, len(currentLoad))
	for k, v := range currentLoad {
		load[k] = int32(v)
	}

	ctx, cancel := context.WithTimeout(context.Background(), heartbeatTimeout)
	defer cancel()

	resp, err := c.client.Heartbeat(ctx, &proto.HeartbeatRequest{
		WorkerId:    workerID,
		CurrentLoad: load,
	})
	if err != nil {
		return fmt.Errorf("heartbeat rpc error: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("heartbeat failed: code=%d msg=%s", resp.Code, resp.Message)
	}
	return nil
}

// ReportTaskResult 上报任务执行结果
func (c *EngineClient) ReportTaskResult(result *types.TaskResult) error {
	outputBytes, err := json.Marshal(result.OutputData)
	if err != nil {
		return fmt.Errorf("marshal output data error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	start := time.Now()
	resp, err := c.client.ReportTaskResult(ctx, &proto.ReportTaskResultRequest{
		TaskId:             result.TaskID,
		WorkflowInstanceId: result.InstanceID,
		WorkflowNodeId:     result.NodeID,
		Success:            result.Success,
		OutputData:         outputBytes,
		ErrorMessage:       result.ErrorMessage,
		StartTime:          result.StartTime.UnixMilli(),
		EndTime:            result.EndTime.UnixMilli(),
	})
	elapsed := time.Since(start)

	if err != nil {
		log.Error().Err(err).
			Str("instance_id", result.InstanceID).
			Str("node_id", result.NodeID).
			Dur("elapsed", elapsed).
			Msg("ReportTaskResult rpc failed")
		return fmt.Errorf("report task result rpc error: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("report task result failed: code=%d msg=%s", resp.Code, resp.Message)
	}
	log.Debug().
		Str("instance_id", result.InstanceID).
		Str("node_id", result.NodeID).
		Bool("success", result.Success).
		Dur("elapsed", elapsed).
		Msg("ReportTaskResult success")
	return nil
}

// ReportError 上报任务执行错误
func (c *EngineClient) ReportError(workerID, workerIP, instanceID, nodeID, errMsg, stackTrace string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	resp, err := c.client.ReportError(ctx, &proto.ReportErrorRequest{
		WorkerId:           workerID,
		WorkerIp:           workerIP,
		WorkflowInstanceId: instanceID,
		WorkflowNodeId:     nodeID,
		ErrorMessage:       errMsg,
		StackTrace:         stackTrace,
		OccurTime:          time.Now().UnixMilli(),
	})
	if err != nil {
		return fmt.Errorf("report error rpc error: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("report error failed: code=%d msg=%s", resp.Code, resp.Message)
	}
	return nil
}

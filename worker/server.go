package worker

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/proto"
	"github.com/zhiyunliu/distributed-workflow/types"
)

// WorkerServer 实现 proto.NodeWorkerServiceServer，接收 Engine 的 gRPC 调用
type WorkerServer struct {
	proto.UnimplementedNodeWorkerServiceServer
	taskMgr interfaces.TaskExecutionManager
}

// NewWorkerServer 创建 gRPC 服务端
func NewWorkerServer(taskMgr interfaces.TaskExecutionManager) *WorkerServer {
	return &WorkerServer{taskMgr: taskMgr}
}

// RegisterServer 将 WorkerServer 注册到 gRPC server
func (s *WorkerServer) RegisterServer(grpcServer *grpc.Server) {
	proto.RegisterNodeWorkerServiceServer(grpcServer, s)
}

// AssignTask 接收 Engine 分配的任务并提交执行
func (s *WorkerServer) AssignTask(_ context.Context, req *proto.AssignTaskRequest) (*proto.AssignTaskResponse, error) {
	var config map[string]interface{}
	if len(req.Configuration) > 0 {
		if err := json.Unmarshal(req.Configuration, &config); err != nil {
			log.Warn().Err(err).Str("task_id", req.TaskId).Msg("unmarshal configuration failed")
		}
	}
	var inputData map[string]interface{}
	if len(req.InputData) > 0 {
		if err := json.Unmarshal(req.InputData, &inputData); err != nil {
			log.Warn().Err(err).Str("task_id", req.TaskId).Msg("unmarshal input_data failed")
		}
	}

	task := &types.Task{
		TaskID:     req.TaskId,
		InstanceID: req.WorkflowInstanceId,
		NodeID:     req.WorkflowNodeId,
		NodeType:   req.NodeType,
		Config:     config,
		InputData:  inputData,
		Timeout:    int(req.Timeout),
	}

	if err := s.taskMgr.Submit(task); err != nil {
		log.Error().Err(err).Str("task_id", req.TaskId).Msg("submit task failed")
		return &proto.AssignTaskResponse{Code: 1, Message: err.Error(), Success: false}, nil
	}
	return &proto.AssignTaskResponse{Code: 0, Message: "ok", Success: true}, nil
}

// CancelTask 取消正在执行的任务
func (s *WorkerServer) CancelTask(_ context.Context, req *proto.CancelTaskRequest) (*proto.CancelTaskResponse, error) {
	if err := s.taskMgr.Cancel(req.WorkflowInstanceId, req.WorkflowNodeId); err != nil {
		log.Warn().Err(err).
			Str("instance_id", req.WorkflowInstanceId).
			Str("node_id", req.WorkflowNodeId).
			Msg("cancel task failed")
		return &proto.CancelTaskResponse{Code: 1, Message: err.Error(), Success: false}, nil
	}
	return &proto.CancelTaskResponse{Code: 0, Message: "ok", Success: true}, nil
}

// PauseTask 暂停正在执行的任务（D2 新增）
func (s *WorkerServer) PauseTask(_ context.Context, req *proto.PauseTaskRequest) (*proto.PauseTaskResponse, error) {
	if err := s.taskMgr.Cancel(req.WorkflowInstanceId, req.WorkflowNodeId); err != nil {
		log.Warn().Err(err).
			Str("instance_id", req.WorkflowInstanceId).
			Str("node_id", req.WorkflowNodeId).
			Msg("pause task failed")
		return &proto.PauseTaskResponse{Code: 1, Message: err.Error(), Success: false}, nil
	}
	log.Info().
		Str("instance_id", req.WorkflowInstanceId).
		Str("node_id", req.WorkflowNodeId).
		Msg("task paused")
	return &proto.PauseTaskResponse{Code: 0, Message: "ok", Success: true}, nil
}

// HealthCheck Worker 健康检查（D2 新增）
func (s *WorkerServer) HealthCheck(_ context.Context, req *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	log.Debug().Str("worker_id", req.WorkerId).Msg("health check received")
	return &proto.HealthCheckResponse{Code: 0, Message: "ok", Success: true}, nil
}

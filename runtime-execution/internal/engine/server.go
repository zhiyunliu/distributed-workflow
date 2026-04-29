package engine

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/proto"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// EngineServer 实现 proto.EngineServiceServer，接收 Worker 的 gRPC 调用
type EngineServer struct {
	proto.UnimplementedEngineServiceServer
	workerMgr api.WorkerManagerService
	state     api.InstanceStateService
}

// NewEngineServer 创建 gRPC 服务端
func NewEngineServer(
	workerMgr api.WorkerManagerService,
	state api.InstanceStateService,
) *EngineServer {
	return &EngineServer{workerMgr: workerMgr, state: state}
}

// RegisterServer 将 EngineServer 注册到 gRPC server
func (s *EngineServer) RegisterServer(grpcServer *grpc.Server) {
	proto.RegisterEngineServiceServer(grpcServer, s)
}

// RegisterWorker Worker 注册
func (s *EngineServer) RegisterWorker(_ context.Context, req *proto.RegisterWorkerRequest) (*proto.RegisterWorkerResponse, error) {
	caps := make(map[string]int, len(req.Capabilities))
	for k, v := range req.Capabilities {
		caps[k] = int(v)
	}
	info := &types.NodeWorkerInfo{
		ID:            req.WorkerId,
		Address:       req.Address,
		IP:            req.WorkerIp,
		Capabilities:  caps,
		CurrentLoad:   make(map[string]int),
		LastHeartbeat: time.Now(),
		Status:        types.WorkerStatusOnline,
	}
	if err := s.workerMgr.RegisterWorker(info); err != nil {
		log.Error().Err(err).Str("worker_id", req.WorkerId).Msg("register worker failed")
		return &proto.RegisterWorkerResponse{Code: 1, Message: err.Error(), Success: false}, nil
	}
	return &proto.RegisterWorkerResponse{Code: 0, Message: "ok", Success: true}, nil
}

// UnregisterWorker Worker 注销
func (s *EngineServer) UnregisterWorker(_ context.Context, req *proto.UnregisterWorkerRequest) (*proto.UnregisterWorkerResponse, error) {
	if err := s.workerMgr.HandleWorkerOffline(req.WorkerId); err != nil {
		return &proto.UnregisterWorkerResponse{Code: 1, Message: err.Error(), Success: false}, nil
	}
	log.Info().Str("worker_id", req.WorkerId).Msg("worker unregistered")
	return &proto.UnregisterWorkerResponse{Code: 0, Message: "ok", Success: true}, nil
}

// Heartbeat Worker 心跳上报
func (s *EngineServer) Heartbeat(_ context.Context, req *proto.HeartbeatRequest) (*proto.HeartbeatResponse, error) {
	load := make(map[string]int, len(req.CurrentLoad))
	for k, v := range req.CurrentLoad {
		load[k] = int(v)
	}
	if err := s.workerMgr.UpdateHeartbeat(req.WorkerId, load); err != nil {
		log.Warn().Err(err).Str("worker_id", req.WorkerId).Msg("heartbeat update failed")
		return &proto.HeartbeatResponse{Code: 1, Message: err.Error(), Success: false}, nil
	}
	return &proto.HeartbeatResponse{Code: 0, Message: "ok", Success: true}, nil
}

// ReportTaskResult 上报任务执行结果
func (s *EngineServer) ReportTaskResult(_ context.Context, req *proto.ReportTaskResultRequest) (*proto.ReportTaskResultResponse, error) {
	var outputData map[string]interface{}
	if len(req.OutputData) > 0 {
		if err := json.Unmarshal(req.OutputData, &outputData); err != nil {
			log.Warn().Err(err).Str("task_id", req.TaskId).Msg("unmarshal output data failed")
		}
	}

	result := &types.TaskResult{
		TaskID:       req.TaskId,
		InstanceID:   req.WorkflowInstanceId,
		NodeID:       req.WorkflowNodeId,
		Success:      req.Success,
		OutputData:   outputData,
		ErrorMessage: req.ErrorMessage,
		StartTime:    time.UnixMilli(req.StartTime),
		EndTime:      time.UnixMilli(req.EndTime),
	}

	if err := s.state.ReportTaskResult(result); err != nil {
		log.Error().Err(err).Str("task_id", req.TaskId).Msg("report task result failed")
		return &proto.ReportTaskResultResponse{Code: 1, Message: err.Error(), Success: false}, nil
	}
	return &proto.ReportTaskResultResponse{Code: 0, Message: "ok", Success: true}, nil
}

// ReportError 上报执行错误（不影响流程，仅记录日志）
func (s *EngineServer) ReportError(_ context.Context, req *proto.ReportErrorRequest) (*proto.ReportErrorResponse, error) {
	log.Error().
		Str("worker_id", req.WorkerId).
		Str("worker_ip", req.WorkerIp).
		Str("instance_id", req.WorkflowInstanceId).
		Str("node_id", req.WorkflowNodeId).
		Str("error", req.ErrorMessage).
		Str("stack", req.StackTrace).
		Int64("occur_time", req.OccurTime).
		Msg("worker reported error")
	return &proto.ReportErrorResponse{Code: 0, Message: "ok", Success: true}, nil
}



package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/node"
)

// taskHandle 运行中的任务句柄
type taskHandle struct {
	cancel context.CancelFunc
}

// TaskManager 实现 api.TaskExecutionManager
// 按节点类型提供并发限制（semaphore），每个任务独立 goroutine 执行
type TaskManager struct {
	container api.NodeExecutorContainer
	engineRPC TaskResultReporter
	mu        sync.Mutex
	running   map[string]*taskHandle // key: instanceID+"-"+nodeID
	caps      map[string]int         // 每类型最大并发数
	loadMu    sync.RWMutex
	load      map[string]int // 当前各类型执行数
}

// TaskResultReporter 上报任务结果的接口（由 rpc.EngineClient 实现）
type TaskResultReporter interface {
	ReportTaskResult(result *types.TaskResult) error
}

// NewTaskManager 创建任务执行管理器
// caps: 每个节点类型的最大并发数，例如 {"log":10, "http":5}
func NewTaskManager(
	container api.NodeExecutorContainer,
	reporter TaskResultReporter,
	caps map[string]int,
) api.TaskExecutionManager {
	return &TaskManager{
		container: container,
		engineRPC: reporter,
		running:   make(map[string]*taskHandle),
		caps:      caps,
		load:      make(map[string]int),
	}
}

func (m *TaskManager) Submit(task *types.Task) error {
	key := task.InstanceID + "-" + task.NodeID

	// 并发数检查
	m.loadMu.RLock()
	cur := m.load[task.NodeType]
	cap := m.caps[task.NodeType]
	m.loadMu.RUnlock()

	if cap > 0 && cur >= cap {
		return fmt.Errorf("node type '%s' at full capacity (%d/%d)", task.NodeType, cur, cap)
	}

	executor, err := m.container.Get(task.NodeType)
	if err != nil {
		return fmt.Errorf("get executor for type '%s': %w", task.NodeType, err)
	}

	// 注册运行句柄
	ctx, cancel := context.WithCancel(context.Background())
	if task.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(task.Timeout)*time.Second)
	}

	m.mu.Lock()
	m.running[key] = &taskHandle{cancel: cancel}
	m.mu.Unlock()

	m.incrLoad(task.NodeType)

	go m.runTask(ctx, cancel, key, task, executor)
	return nil
}

func (m *TaskManager) Cancel(instanceID string, nodeID string) error {
	key := instanceID + "-" + nodeID
	m.mu.Lock()
	h, ok := m.running[key]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("task '%s' not running", key)
	}
	h.cancel()
	return nil
}

func (m *TaskManager) GetCurrentLoad() map[string]int {
	m.loadMu.RLock()
	defer m.loadMu.RUnlock()
	cp := make(map[string]int, len(m.load))
	for k, v := range m.load {
		cp[k] = v
	}
	return cp
}

func (m *TaskManager) runTask(ctx context.Context, cancel context.CancelFunc, key string, task *types.Task, executor api.NodeExecutor) {
	defer cancel()
	defer func() {
		m.mu.Lock()
		delete(m.running, key)
		m.mu.Unlock()
		m.decrLoad(task.NodeType)
	}()

	startTime := time.Now()
	wfCtx := &types.WorkflowContext{
		InstanceID: task.InstanceID,
		Data:       task.InputData,
	}

	// 在独立 goroutine 执行，监听 ctx.Done() 取消
	type execResult struct {
		output map[string]interface{}
		err    error
	}
	resultCh := make(chan execResult, 1)
	go func() {
		output, err := executor.Execute(task.Config, task.InputData, wfCtx)
		resultCh <- execResult{output: output, err: err}
	}()

	var output map[string]interface{}
	var execErr error

	select {
	case <-ctx.Done():
		execErr = fmt.Errorf("task '%s' cancelled or timed out: %w", key, ctx.Err())
	case res := <-resultCh:
		output = res.output
		execErr = res.err
	}

	endTime := time.Now()
	errMsg := ""

	// ErrNodeWaiting 是人工任务进入等待状态的哨兵错误，
	// 需通知引擎将节点标记为 waiting，而非失败。
	if errors.Is(execErr, node.ErrNodeWaiting) {
		log.Info().
			Str("instance_id", task.InstanceID).
			Str("node_id", task.NodeID).
			Msg("task entered human approval waiting state")
		result := &types.TaskResult{
			TaskID:       task.TaskID,
			InstanceID:   task.InstanceID,
			NodeID:       task.NodeID,
			Success:      false,
			OutputData:   nil,
			ErrorMessage: node.ErrNodeWaiting.Error(),
			StartTime:    startTime,
			EndTime:      endTime,
		}
		if err := m.engineRPC.ReportTaskResult(result); err != nil {
			log.Error().Err(err).Str("task_id", task.TaskID).Msg("report waiting task result failed")
		}
		return
	}

	if execErr != nil {
		errMsg = execErr.Error()
		log.Error().Err(execErr).
			Str("instance_id", task.InstanceID).
			Str("node_id", task.NodeID).
			Msg("task execution failed")
	}

	result := &types.TaskResult{
		TaskID:       task.TaskID,
		InstanceID:   task.InstanceID,
		NodeID:       task.NodeID,
		Success:      execErr == nil,
		OutputData:   output,
		ErrorMessage: errMsg,
		StartTime:    startTime,
		EndTime:      endTime,
	}

	if err := m.engineRPC.ReportTaskResult(result); err != nil {
		log.Error().Err(err).
			Str("task_id", task.TaskID).
			Msg("report task result failed")
	}
}

func (m *TaskManager) incrLoad(nodeType string) {
	m.loadMu.Lock()
	m.load[nodeType]++
	m.loadMu.Unlock()
}

func (m *TaskManager) decrLoad(nodeType string) {
	m.loadMu.Lock()
	if m.load[nodeType] > 0 {
		m.load[nodeType]--
	}
	m.loadMu.Unlock()
}

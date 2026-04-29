package node

import (
	"errors"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// ErrNodeWaiting 是一个哨兵错误，用于标识节点进入人工审批等待状态。
// 调度器检测到此错误时，应将节点状态更新为 waiting/approvalStatus=pending，而不是按失败处理。
var ErrNodeWaiting = errors.New("node is waiting for human approval")

// HumanTaskExecutor 人工任务执行器
// 节点类型 "human_task" 不做实际计算，直接返回 ErrNodeWaiting，
// 由调度层负责持久化等待状态并等待外部审批动作驱动继续。
//
// 配置字段（均可选，实际由审批服务读取，执行器不解析）：
//   - approvers       []string  审批人列表
//   - approvalMode    string    审批模式 single/or_sign/sequential
//   - title           string    任务标题
//   - description     string    任务描述
//   - formFields      []object  表单字段定义
type HumanTaskExecutor struct{}

// Type 返回执行器类型标识
func (e *HumanTaskExecutor) Type() string { return "human_task" }

// Execute 返回 ErrNodeWaiting，触发调度层进入等待逻辑
func (e *HumanTaskExecutor) Execute(config map[string]interface{}, input map[string]interface{}, ctx *types.WorkflowContext) (map[string]interface{}, error) {
	return nil, ErrNodeWaiting
}



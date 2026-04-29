// Package types 定义了 DistributedWorkflow 工作流引擎的所有核心数据结构与枚举类型。
// 本包是整个引擎的基础类型层，不依赖任何外部包，仅使用 Go 标准库。
//
// 核心类型分为三类：
//  1. 枚举类型：WorkflowStatus、WorkflowNodeStatus、ConnectionType、DependencyMode
//  2. 流程定义类型：WorkflowDef、WorkflowNode、Connection
//  3. 运行时类型：WorkflowInstance、WorkflowNodeState、WorkflowContext、NodeWorkerInfo、Task、TaskResult
package types


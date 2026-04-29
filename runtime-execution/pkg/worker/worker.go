package worker

import (
	internalworker "github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/worker"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
)

// Config 是对外暴露的 worker 配置。
type Config = internalworker.Config

// NodeWorker 是对外暴露的 worker 类型别名。
type NodeWorker = internalworker.NodeWorker

// New 创建 worker。
func New(cfg Config, container api.NodeExecutorContainer) (*NodeWorker, error) {
	return internalworker.New(cfg, container)
}

// NewExecutorContainer 创建执行器容器。
func NewExecutorContainer() api.NodeExecutorContainer {
	return internalworker.NewExecutorContainer()
}

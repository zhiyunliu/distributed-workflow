package worker

import (
	"fmt"
	"sync"

	"github.com/zhiyunliu/distributed-workflow/interfaces"
)

// ExecutorContainer 实现 interfaces.NodeExecutorContainer
type ExecutorContainer struct {
	mu        sync.RWMutex
	executors map[string]interfaces.NodeExecutor
}

// NewExecutorContainer 创建执行器容器
func NewExecutorContainer() interfaces.NodeExecutorContainer {
	return &ExecutorContainer{
		executors: make(map[string]interfaces.NodeExecutor),
	}
}

func (c *ExecutorContainer) Register(executor interfaces.NodeExecutor) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := executor.Type()
	if _, exists := c.executors[t]; exists {
		return fmt.Errorf("executor type '%s' already registered", t)
	}
	c.executors[t] = executor
	return nil
}

func (c *ExecutorContainer) Get(nodeType string) (interfaces.NodeExecutor, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.executors[nodeType]
	if !ok {
		return nil, fmt.Errorf("executor type '%s' not found", nodeType)
	}
	return e, nil
}

func (c *ExecutorContainer) List() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	types := make([]string, 0, len(c.executors))
	for t := range c.executors {
		types = append(types, t)
	}
	return types
}

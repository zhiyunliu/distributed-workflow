package worker

import (
	"testing"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
)

// stubExecutor 用于测试的桩执行器
type stubExecutor struct{ t string }

func (s *stubExecutor) Type() string { return s.t }
func (s *stubExecutor) Execute(config map[string]interface{}, input map[string]interface{}, ctx *types.WorkflowContext) (map[string]interface{}, error) {
	return map[string]interface{}{"stub": true}, nil
}

func TestExecutorContainer_RegisterAndGet(t *testing.T) {
	c := NewExecutorContainer()

	err := c.Register(&stubExecutor{t: "foo"})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	exec, err := c.Get("foo")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if exec.Type() != "foo" {
		t.Errorf("expected type 'foo', got %q", exec.Type())
	}
}

func TestExecutorContainer_DuplicateRegister(t *testing.T) {
	c := NewExecutorContainer()
	_ = c.Register(&stubExecutor{t: "bar"})

	err := c.Register(&stubExecutor{t: "bar"})
	if err == nil {
		t.Error("expected error on duplicate registration, got nil")
	}
}

func TestExecutorContainer_GetNotFound(t *testing.T) {
	c := NewExecutorContainer()

	_, err := c.Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown type, got nil")
	}
}

func TestExecutorContainer_List(t *testing.T) {
	c := NewExecutorContainer()
	_ = c.Register(&stubExecutor{t: "a"})
	_ = c.Register(&stubExecutor{t: "b"})

	list := c.List()
	if len(list) != 2 {
		t.Errorf("expected 2 executors, got %d", len(list))
	}
	m := make(map[string]bool)
	for _, v := range list {
		m[v] = true
	}
	if !m["a"] || !m["b"] {
		t.Errorf("expected 'a' and 'b' in list, got %v", list)
	}
}

func TestExecutorContainer_ImplementsInterface(t *testing.T) {
	c := NewExecutorContainer()
	var _ api.NodeExecutorContainer = c
}



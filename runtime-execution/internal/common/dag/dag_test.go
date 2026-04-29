package dag

import (
	"testing"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// buildDef 构建工作流定义辅助函数
func buildDef(nodes map[string]*types.WorkflowNode, conns []*types.Connection, start, end string) *types.WorkflowDef {
	return &types.WorkflowDef{
		ID:          "test-workflow",
		Name:        "Test Workflow",
		Nodes:       nodes,
		Connections: conns,
		StartNodeID: start,
		EndNodeID:   end,
	}
}

func node(id, nodeType string, depMode types.DependencyMode) *types.WorkflowNode {
	return &types.WorkflowNode{ID: id, Type: nodeType, Name: id, DependencyMode: depMode}
}

func conn(id, src, tgt string, connType types.ConnectionType) *types.Connection {
	return &types.Connection{ID: id, SourceID: src, TargetID: tgt, Type: connType}
}

func nodeState(nodeID string, status types.WorkflowNodeStatus) *types.WorkflowNodeState {
	return &types.WorkflowNodeState{NodeID: nodeID, Status: status}
}

// ─── 循环检测测试 ─────────────────────────────────────────────────────────

func TestHasCycle_NoCycle_Serial(t *testing.T) {
	// A → B → C
	def := buildDef(
		map[string]*types.WorkflowNode{
			"A": node("A", "log", ""),
			"B": node("B", "log", ""),
			"C": node("C", "log", ""),
		},
		[]*types.Connection{
			conn("c1", "A", "B", types.ConnectionTypeSuccess),
			conn("c2", "B", "C", types.ConnectionTypeSuccess),
		},
		"A", "C",
	)
	parser := NewDAGParser()
	g, err := parser.Parse(def)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.HasCycle() {
		t.Error("expected no cycle in serial DAG")
	}
}

func TestHasCycle_WithCycle(t *testing.T) {
	// A → B → C → A（循环）
	def := &types.WorkflowDef{
		ID:          "cyclic",
		Name:        "Cyclic",
		StartNodeID: "A",
		EndNodeID:   "C",
		Nodes: map[string]*types.WorkflowNode{
			"A": node("A", "log", ""),
			"B": node("B", "log", ""),
			"C": node("C", "log", ""),
		},
		Connections: []*types.Connection{
			conn("c1", "A", "B", types.ConnectionTypeSuccess),
			conn("c2", "B", "C", types.ConnectionTypeSuccess),
			conn("c3", "C", "A", types.ConnectionTypeSuccess), // 循环
		},
	}

	// 绕过 Parse 校验直接测试 HasCycle
	g := &DAGGraph{
		nodes:       def.Nodes,
		connections: def.Connections,
	}
	if !g.HasCycle() {
		t.Error("expected cycle to be detected")
	}
}

func TestParse_CircularDependency(t *testing.T) {
	// Parse 应该返回错误
	def := &types.WorkflowDef{
		ID:          "cyclic",
		Name:        "Cyclic",
		StartNodeID: "A",
		EndNodeID:   "C",
		Nodes: map[string]*types.WorkflowNode{
			"A": node("A", "log", ""),
			"B": node("B", "log", ""),
			"C": node("C", "log", ""),
		},
		Connections: []*types.Connection{
			conn("c1", "A", "B", types.ConnectionTypeSuccess),
			conn("c2", "B", "C", types.ConnectionTypeSuccess),
			conn("c3", "C", "A", types.ConnectionTypeSuccess),
		},
	}
	_, err := NewDAGParser().Parse(def)
	if err == nil {
		t.Error("expected error for circular dependency")
	}
}

// ─── All/Any 依赖模式测试 ─────────────────────────────────────────────────

func TestGetReadyNodes_AllMode(t *testing.T) {
	// A → C (all), B → C (all): C 需要 A 和 B 都完成
	def := buildDef(
		map[string]*types.WorkflowNode{
			"A": node("A", "log", types.DependencyModeAll),
			"B": node("B", "log", types.DependencyModeAll),
			"C": node("C", "log", types.DependencyModeAll),
		},
		[]*types.Connection{
			conn("c1", "A", "C", types.ConnectionTypeSuccess),
			conn("c2", "B", "C", types.ConnectionTypeSuccess),
		},
		"A", "C",
	)
	parser := NewDAGParser()
	g, err := parser.Parse(def)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// 初始状态：只有 A 和 B 可以执行（无前置节点）
	states := map[string]*types.WorkflowNodeState{}
	ready := g.GetReadyNodes(states)
	if len(ready) != 2 {
		t.Errorf("expected 2 ready nodes (A,B), got %d: %v", len(ready), ready)
	}

	// A 完成，B 未完成 → C 不可调度
	states["A"] = nodeState("A", types.WorkflowNodeStatusCompleted)
	states["B"] = nodeState("B", types.WorkflowNodeStatusPending)
	ready = g.GetReadyNodes(states)
	for _, id := range ready {
		if id == "C" {
			t.Error("C should not be ready when B is still pending (All mode)")
		}
	}

	// A 和 B 都完成 → C 可调度
	states["B"] = nodeState("B", types.WorkflowNodeStatusCompleted)
	ready = g.GetReadyNodes(states)
	found := false
	for _, id := range ready {
		if id == "C" {
			found = true
		}
	}
	if !found {
		t.Errorf("C should be ready when both A and B are completed, got: %v", ready)
	}
}

func TestGetReadyNodes_AnyMode(t *testing.T) {
	// A → C (any), B → C (any): C 只需要 A 或 B 之一完成
	def := buildDef(
		map[string]*types.WorkflowNode{
			"A": node("A", "log", types.DependencyModeAny),
			"B": node("B", "log", types.DependencyModeAny),
			"C": node("C", "log", types.DependencyModeAny),
		},
		[]*types.Connection{
			conn("c1", "A", "C", types.ConnectionTypeSuccess),
			conn("c2", "B", "C", types.ConnectionTypeSuccess),
		},
		"A", "C",
	)
	parser := NewDAGParser()
	g, err := parser.Parse(def)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// A 完成，B 未完成 → C 可调度（Any 模式）
	states := map[string]*types.WorkflowNodeState{
		"A": nodeState("A", types.WorkflowNodeStatusCompleted),
		"B": nodeState("B", types.WorkflowNodeStatusPending),
	}
	ready := g.GetReadyNodes(states)
	found := false
	for _, id := range ready {
		if id == "C" {
			found = true
		}
	}
	if !found {
		t.Errorf("C should be ready in Any mode when A is completed, got: %v", ready)
	}
}

func TestGetReadyNodes_BranchDAG(t *testing.T) {
	// Start → A (success branch) → End
	//       → B (failure branch) → End
	def := buildDef(
		map[string]*types.WorkflowNode{
			"Start": node("Start", "log", types.DependencyModeAll),
			"A":     node("A", "log", types.DependencyModeAll),
			"B":     node("B", "log", types.DependencyModeAll),
			"End":   node("End", "log", types.DependencyModeAny),
		},
		[]*types.Connection{
			conn("c1", "Start", "A", types.ConnectionTypeSuccess),
			conn("c2", "Start", "B", types.ConnectionTypeFailure),
			conn("c3", "A", "End", types.ConnectionTypeSuccess),
			conn("c4", "B", "End", types.ConnectionTypeSuccess),
		},
		"Start", "End",
	)
	parser := NewDAGParser()
	g, err := parser.Parse(def)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Start 完成后，A 可调度（success 连线匹配），B 不可调度（failure 连线不匹配）
	states := map[string]*types.WorkflowNodeState{
		"Start": nodeState("Start", types.WorkflowNodeStatusCompleted),
	}
	ready := g.GetReadyNodes(states)
	hasA, hasB := false, false
	for _, id := range ready {
		if id == "A" {
			hasA = true
		}
		if id == "B" {
			hasB = true
		}
	}
	if !hasA {
		t.Error("A should be ready after Start completes (success branch)")
	}
	if hasB {
		t.Error("B should NOT be ready after Start completes (failure branch requires failure)")
	}
}

func TestParse_MissingStartNode(t *testing.T) {
	def := buildDef(
		map[string]*types.WorkflowNode{"A": node("A", "log", "")},
		[]*types.Connection{},
		"NOT_EXIST", "A",
	)
	_, err := NewDAGParser().Parse(def)
	if err == nil {
		t.Error("expected error for missing start node")
	}
}



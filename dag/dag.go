package dag

import (
	"fmt"

	"github.com/zhiyunliu/distributed-workflow/types"
)

// DAGGraph 表示工作流的有向无环图，包含邻接表与反向邻接表
type DAGGraph struct {
	// nodes 所有节点，key 为节点 ID
	nodes map[string]*types.WorkflowNode
	// connections 所有连线
	connections []*types.Connection
	// successors 正向邻接表：key 为源节点 ID，value 为连线列表
	successors map[string][]*types.Connection
	// predecessors 反向邻接表：key 为目标节点 ID，value 为连线列表
	predecessors map[string][]*types.Connection
	// startNodeID 起始节点 ID
	startNodeID string
	// endNodeID 结束节点 ID
	endNodeID string
}

// DAGParser 工作流 DAG 解析器
type DAGParser struct{}

// NewDAGParser 创建 DAG 解析器
func NewDAGParser() *DAGParser {
	return &DAGParser{}
}

// Parse 将 WorkflowDef 解析为 DAGGraph
func (p *DAGParser) Parse(def *types.WorkflowDef) (*DAGGraph, error) {
	if def == nil {
		return nil, fmt.Errorf("workflow definition is nil")
	}
	if len(def.Nodes) == 0 {
		return nil, fmt.Errorf("workflow definition has no nodes")
	}
	if len(def.Connections) == 0 {
		return nil, fmt.Errorf("workflow definition has no connections")
	}
	if _, ok := def.Nodes[def.StartNodeID]; !ok {
		return nil, fmt.Errorf("start node '%s' not found in nodes", def.StartNodeID)
	}
	if _, ok := def.Nodes[def.EndNodeID]; !ok {
		return nil, fmt.Errorf("end node '%s' not found in nodes", def.EndNodeID)
	}

	g := &DAGGraph{
		nodes:        def.Nodes,
		connections:  def.Connections,
		successors:   make(map[string][]*types.Connection),
		predecessors: make(map[string][]*types.Connection),
		startNodeID:  def.StartNodeID,
		endNodeID:    def.EndNodeID,
	}

	// 构建邻接表
	for _, conn := range def.Connections {
		if _, ok := def.Nodes[conn.SourceID]; !ok {
			return nil, fmt.Errorf("connection '%s' source node '%s' not found", conn.ID, conn.SourceID)
		}
		if _, ok := def.Nodes[conn.TargetID]; !ok {
			return nil, fmt.Errorf("connection '%s' target node '%s' not found", conn.ID, conn.TargetID)
		}
		g.successors[conn.SourceID] = append(g.successors[conn.SourceID], conn)
		g.predecessors[conn.TargetID] = append(g.predecessors[conn.TargetID], conn)
	}

	// 检查循环依赖
	if g.HasCycle() {
		return nil, fmt.Errorf("workflow definition has circular dependency")
	}

	return g, nil
}

// HasCycle 检测 DAG 是否存在循环依赖，使用三色 DFS 算法
// 返回 true 表示存在循环
func (g *DAGGraph) HasCycle() bool {
	// 构建简单邻接表（只需节点 ID 列表）
	adj := make(map[string][]string)
	for _, conn := range g.connections {
		adj[conn.SourceID] = append(adj[conn.SourceID], conn.TargetID)
	}

	// 访问状态：0=未访问，1=正在访问（DFS 路径上），2=已完成访问
	visited := make(map[string]int)

	var dfs func(nodeID string) bool
	dfs = func(nodeID string) bool {
		if visited[nodeID] == 1 {
			// 当前节点在 DFS 路径上被再次访问，发现循环
			return true
		}
		if visited[nodeID] == 2 {
			// 已完成访问，无循环
			return false
		}
		// 标记为正在访问
		visited[nodeID] = 1
		for _, nextID := range adj[nodeID] {
			if dfs(nextID) {
				return true
			}
		}
		// 标记为已完成访问
		visited[nodeID] = 2
		return false
	}

	for nodeID := range g.nodes {
		if dfs(nodeID) {
			return true
		}
	}
	return false
}

// GetPredecessors 返回指定节点的所有前置连线
func (g *DAGGraph) GetPredecessors(nodeID string) []*types.Connection {
	return g.predecessors[nodeID]
}

// GetSuccessors 返回指定节点的所有后继连线
func (g *DAGGraph) GetSuccessors(nodeID string) []*types.Connection {
	return g.successors[nodeID]
}

// GetStartNodeID 返回起始节点 ID
func (g *DAGGraph) GetStartNodeID() string {
	return g.startNodeID
}

// GetEndNodeID 返回结束节点 ID
func (g *DAGGraph) GetEndNodeID() string {
	return g.endNodeID
}

// GetNode 获取节点定义
func (g *DAGGraph) GetNode(nodeID string) (*types.WorkflowNode, bool) {
	n, ok := g.nodes[nodeID]
	return n, ok
}

// GetAllNodeIDs 获取所有节点 ID 列表
func (g *DAGGraph) GetAllNodeIDs() []string {
	ids := make([]string, 0, len(g.nodes))
	for id := range g.nodes {
		ids = append(ids, id)
	}
	return ids
}

// GetReadyNodes 根据当前节点执行状态，返回满足依赖条件的可执行节点 ID 列表
// states: key 为节点 ID，value 为节点执行状态
func (g *DAGGraph) GetReadyNodes(states map[string]*types.WorkflowNodeState) []string {
	var readyNodes []string

	for nodeID, node := range g.nodes {
		// 已有执行状态（非 pending）的节点不再调度
		if state, exists := states[nodeID]; exists {
			if state.Status != types.WorkflowNodeStatusPending {
				continue
			}
		}

		predecessors := g.predecessors[nodeID]
		// 起始节点（无前置依赖）直接可执行
		if len(predecessors) == 0 {
			// 确认还没有对应状态才加入（首次调度）
			if _, exists := states[nodeID]; !exists {
				readyNodes = append(readyNodes, nodeID)
			}
			continue
		}

		// 检查依赖模式
		dependencyMode := node.DependencyMode
		if dependencyMode == "" {
			dependencyMode = types.DependencyModeAll
		}

		if dependencyMode == types.DependencyModeAll {
			if g.isAllDependencySatisfied(predecessors, states) {
				readyNodes = append(readyNodes, nodeID)
			}
		} else if dependencyMode == types.DependencyModeAny {
			if g.isAnyDependencySatisfied(predecessors, states) {
				readyNodes = append(readyNodes, nodeID)
			}
		}
	}

	return readyNodes
}

// isAllDependencySatisfied 检查 All 模式依赖是否满足
// 所有前置节点必须执行完成（completed/failed），且连线类型与节点状态匹配
func (g *DAGGraph) isAllDependencySatisfied(predecessors []*types.Connection, states map[string]*types.WorkflowNodeState) bool {
	for _, conn := range predecessors {
		srcState, exists := states[conn.SourceID]
		if !exists {
			return false
		}
		if !isConnectionSatisfied(conn.Type, srcState.Status) {
			return false
		}
	}
	return true
}

// isAnyDependencySatisfied 检查 Any 模式依赖是否满足
// 任意一个前置节点满足连线条件即可
func (g *DAGGraph) isAnyDependencySatisfied(predecessors []*types.Connection, states map[string]*types.WorkflowNodeState) bool {
	for _, conn := range predecessors {
		srcState, exists := states[conn.SourceID]
		if !exists {
			continue
		}
		if isConnectionSatisfied(conn.Type, srcState.Status) {
			return true
		}
	}
	return false
}

// isConnectionSatisfied 判断连线条件是否满足
func isConnectionSatisfied(connType types.ConnectionType, srcStatus types.WorkflowNodeStatus) bool {
	switch connType {
	case types.ConnectionTypeSuccess:
		return srcStatus == types.WorkflowNodeStatusCompleted
	case types.ConnectionTypeFailure:
		return srcStatus == types.WorkflowNodeStatusFailed
	case types.ConnectionTypeAlways:
		return srcStatus == types.WorkflowNodeStatusCompleted || srcStatus == types.WorkflowNodeStatusFailed
	default:
		// 默认按 success 处理
		return srcStatus == types.WorkflowNodeStatusCompleted
	}
}

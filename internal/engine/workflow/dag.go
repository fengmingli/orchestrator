package workflow

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DAG 整个图
type DAG struct {
	nodes     map[ID]*Node
	mu        sync.RWMutex
	failed    atomic.Bool        // 全局失败标记
	cancelAll context.CancelFunc // cancelAll 取消所有节点
	// 缓存优化
	topologyCache []ID        // 缓存的拓扑序
	topologyValid atomic.Bool // 拓扑序缓存是否有效
	layersCache   [][]ID      // 缓存的分层结果
	layersValid   atomic.Bool // 分层缓存是否有效
}

// NewDAG 从描述构建，返回环的具体路径（方便前端高亮）
func NewDAG(desc []Desc) (*DAG, error) {
	_, cancel := context.WithCancel(context.Background())
	d := &DAG{
		nodes:     make(map[ID]*Node, len(desc)),
		failed:    atomic.Bool{},
		cancelAll: cancel,
	}
	// 1. 建节点
	for _, n := range desc {
		if _, exists := d.nodes[n.ID]; exists {
			return nil, fmt.Errorf("node %s duplicated", n.ID)
		}
		d.nodes[n.ID] = &Node{
			ID:           n.ID,
			Mode:         n.Mode,
			Runner:       n.Runner,
			Successors:   make(map[ID]*Node),
			Predecessors: make(map[ID]*Node),
			doneCh:       make(chan struct{}),
			policy:       n.Policy,
		}
	}
	// 2. 再建边（依赖 = 父 → 子）
	for _, dsc := range desc {
		to := d.nodes[dsc.ID] // 自己（子）
		for _, fromID := range dsc.Deps {
			if strings.TrimSpace(fromID) == "" {
				continue
			}
			if fromID == dsc.ID { //构建时检测自环
				return nil, fmt.Errorf("self-loop detected: node %s depends on itself", dsc.ID)
			}
			from, ok := d.nodes[fromID]
			if !ok {
				return nil, fmt.Errorf("dependency %s not found", fromID)
			}
			// 正确方向：父 → 子
			from.Successors[dsc.ID] = to
			to.Predecessors[fromID] = from
		}
	}
	// 3. 环检测（Kahn）
	if cycle := d.findCycle(); cycle != nil {
		return nil, fmt.Errorf("cycle detected: %v", cycle)
	}
	// 初始化缓存
	d.invalidateCache()
	return d, nil
}

// Kahn 算法返回环的节点序列（若无环返回 nil）
func (d *DAG) findCycle() []ID {
	d.mu.RLock() // 👈 只读操作，用 RLock 更高效
	defer d.mu.RUnlock()
	return d.findCycleLocked() // 委托给已加锁版本
}

// AddNode 运行时加节点（必须无环）
func (d *DAG) AddNode(n *Node) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.nodes[n.ID]; ok {
		return fmt.Errorf("node %s exists", n.ID)
	}
	//防御性初始化
	if n.doneCh == nil {
		n.doneCh = make(chan struct{})
	}
	d.nodes[n.ID] = n
	if cycle := d.findCycleLocked(); cycle != nil {
		delete(d.nodes, n.ID)
		return fmt.Errorf("add node introduces cycle: %v", cycle)
	}
	// 添加节点后，使缓存失效
	d.invalidateCache()
	return nil
}

// AddEdge 加边（双向维护）
func (d *DAG) AddEdge(from, to ID) error {
	if from == to { // 禁止自环
		return fmt.Errorf("self-loop not allowed:%s -> %s", from, to)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	f, ok1 := d.nodes[from]
	t, ok2 := d.nodes[to]
	if !ok1 || !ok2 {
		return fmt.Errorf("node not found")
	}
	// from !=to ,所以 f和t 是不同的节点，可以安全枷锁
	f.mu.Lock()
	t.mu.Lock()
	defer f.mu.Unlock()
	defer t.mu.Unlock()
	f.Successors[to] = t
	t.Predecessors[from] = f
	if cycle := d.findCycleLocked(); cycle != nil {
		delete(f.Successors, to)
		delete(t.Predecessors, from)
		return fmt.Errorf("add edge introduces cycle: %v", cycle)
	}
	// 添加边后，使缓存失效
	d.invalidateCache()
	return nil
}

// ToGraphviz 导出 DOT
func (d *DAG) ToGraphviz() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	b := &strings.Builder{}
	b.WriteString("digraph G {\n")
	for _, n := range d.nodes {
		shape := map[RunMode]string{
			RunModeSerial:   "box",
			RunModeParallel: "ellipse",
			RunModeMixed:    "diamond",
		}[n.Mode]
		b.WriteString("  ")
		b.WriteString(n.ID)
		b.WriteString(" [label=\"")
		b.WriteString(n.ID)
		b.WriteString("\",shape=")
		b.WriteString(shape)
		b.WriteString("]; \n")
		for to := range n.Successors {
			b.WriteString("  ")
			b.WriteString(n.ID)
			b.WriteString(" -> ")
			b.WriteString(to)
			b.WriteString(";\n")
		}
	}
	b.WriteString("}\n")
	return b.String()
}

// ToMermaid 导出 Mermaid flowchart
func (d *DAG) ToMermaid() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	b := &strings.Builder{}
	b.WriteString("graph TD;\n")
	for _, n := range d.nodes {
		for to := range n.Successors {
			b.WriteString(n.ID)
			b.WriteString(" --> ")
			b.WriteString(to)
			b.WriteString(";\n")
		}
	}
	return b.String()
}

// Snapshot 返回当前已结束节点
func (d *DAG) Snapshot() map[ID]State {
	d.mu.RLock()
	defer d.mu.RUnlock()
	m := make(map[ID]State, len(d.nodes))
	for id, n := range d.nodes {
		m[id] = State(n.State.Load())
	}
	return m
}

// LoadSnapshot 恢复状态（重启后）
func (d *DAG) LoadSnapshot(snap map[ID]State) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for id, state := range snap {
		if n, ok := d.nodes[id]; ok {
			n.State.Store(uint32(state))
		}
	}
}

// ======== 新增辅助函数：执行 Kahn 算法，返回拓扑序和是否成功 ========
// 在已持有 d.mu.RLock() 或 d.mu.Lock() 的情况下调用
func (d *DAG) kahnSortLocked() ([]ID, bool) {
	inDegree := make(map[ID]int, len(d.nodes))
	for id := range d.nodes {
		inDegree[id] = 0
	}
	// 统计入度
	for _, n := range d.nodes {
		for to := range n.Successors {
			inDegree[to]++
		}
	}
	// 初始化队列（入度为0的节点）
	q := make([]ID, 0, len(d.nodes))
	for id, deg := range inDegree {
		if deg == 0 {
			q = append(q, id)
		}
	}
	// Kahn 主循环
	order := make([]ID, 0, len(d.nodes))
	cnt := 0
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		order = append(order, cur)
		cnt++
		for to := range d.nodes[cur].Successors {
			inDegree[to]--
			if inDegree[to] == 0 {
				q = append(q, to)
			}
		}
	}
	// 成功当且仅当处理了所有节点
	return order, cnt == len(d.nodes)
}

// findCycleLocked 查找是否存在环。若有环，返回环序列（从叶子到根）
func (d *DAG) findCycleLocked() []ID {
	_, ok := d.kahnSortLocked()
	if ok {
		return nil // 无环
	}
	// 有环，用 DFS 找出具体环
	visited := make(map[ID]bool, len(d.nodes))
	onStack := make(map[ID]bool, len(d.nodes))
	var cycle []ID
	var dfs func(ID) bool
	dfs = func(id ID) bool {
		visited[id] = true
		onStack[id] = true
		cycle = append(cycle, id)
		for to := range d.nodes[id].Successors {
			//显示检测自环
			if to == id {
				cycle = []ID{id}
				return true
			}
			if !visited[to] {
				if dfs(to) {
					return true
				}
			} else if onStack[to] {
				// 找到环起点，截取
				for i, x := range cycle {
					if x == to {
						cycle = cycle[i:]
						return true
					}
				}
			}
		}
		// 回溯
		cycle = cycle[:len(cycle)-1]
		onStack[id] = false
		return false
	}
	for id := range d.nodes {
		if !visited[id] {
			if dfs(id) {
				return cycle
			}
		}
	}
	return nil
}

// TopoSort ======== 新增：TopoSort 返回拓扑序 ========
// 线程安全，只读操作
func (d *DAG) TopoSort() ([]ID, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	order, ok := d.kahnSortLocked()
	if !ok {
		return nil, fmt.Errorf("graph contains cycle, cannot perform topological sort")
	}
	return order, nil
}

// RemoveNode 删除节点及其所有相关边，并强制保证 DAG 无环
// 如果节点不存在，返回 error
// 删除后会重新检测整个图是否无环，确保强一致性
func (d *DAG) RemoveNode(id ID) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	node, ok := d.nodes[id]
	if !ok {
		return fmt.Errorf("node %s not found", id)
	}
	// 1. 从所有前驱节点的 Successors 中移除自己
	for predID := range node.Predecessors {
		if pred, ok := d.nodes[predID]; ok {
			delete(pred.Successors, id)
		}
	}
	// 2. 从所有后继节点的 Predecessors 中移除自己
	for succID := range node.Successors {
		if succ, ok := d.nodes[succID]; ok {
			delete(succ.Predecessors, id)
		}
	}
	// 3. 从图中删除自己
	delete(d.nodes, id)
	// 4. 强保证：删除后重新检测环！
	//  虽然删除不会引入环，但为了系统强一致性，必须验证！
	if cycle := d.findCycleLocked(); cycle != nil {
		// 理论上不应该发生！如果发生，说明图结构有隐藏 bug
		return fmt.Errorf("internal error: DAG has cycle after removing node %s: %v", id, cycle)
	}
	// 删除节点后，使缓存失效
	d.invalidateCache()
	return nil
}

// SSnapshot 生成当前 DAG 的结构快照
func (d *DAG) SSnapshot() DAGSnapshot {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var nodes []NodeSnapshot
	for _, node := range d.nodes {
		deps := make([]ID, 0, len(node.Predecessors))
		for depID := range node.Predecessors {
			deps = append(deps, depID)
		}
		// 保持顺序一致，便于对比
		sort.Slice(deps, func(i, j int) bool {
			return deps[i] < deps[j]
		})
		nodes = append(nodes, NodeSnapshot{
			ID:   node.ID,
			Deps: deps,
			Mode: node.Mode,
		})
	}
	// 保持节点顺序一致
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
	//只需要当前的Node的结构信息
	return DAGSnapshot{
		Nodes: nodes,
	}
}

// GetStructuralMD5 为 DAG 生成结构 MD5 版本号
func (d *DAG) GetStructuralMD5() (string, error) {
	snap := d.SSnapshot() // 使用之前定义的 Snapshot
	// 序列化为 JSON（确保结构稳定）
	data, err := json.Marshal(snap)
	if err != nil {
		return "", err
	}
	// 计算 MD5，附带时间戳（秒）
	hash := md5.Sum(data)
	return fmt.Sprintf("%d-%x", time.Now().Unix(), hash), nil
}

// invalidateCache 使所有缓存失效
func (d *DAG) invalidateCache() {
	d.topologyValid.Store(false)
	d.layersValid.Store(false)
}

// TopoSortCached 返回缓存的拓扑序，如果缓存无效则重新计算
func (d *DAG) TopoSortCached() ([]ID, error) {
	// 先检查缓存是否有效
	if d.topologyValid.Load() {
		return d.topologyCache, nil
	}

	// 缓存无效，重新计算
	d.mu.Lock()
	defer d.mu.Unlock()

	// 双重检查，避免多个goroutine同时重新计算
	if d.topologyValid.Load() {
		return d.topologyCache, nil
	}

	order, ok := d.kahnSortLocked()
	if !ok {
		return nil, fmt.Errorf("graph contains cycle, cannot perform topological sort")
	}

	// 更新缓存
	d.topologyCache = order
	d.topologyValid.Store(true)

	return order, nil
}

// topoLayersCached 返回缓存的分层结果，如果缓存无效则重新计算
func (d *DAG) topoLayersCached() ([][]ID, error) {
	// 先检查缓存是否有效
	if d.layersValid.Load() {
		return d.layersCache, nil
	}

	// 缓存无效，重新计算
	d.mu.Lock()
	defer d.mu.Unlock()

	// 双重检查
	if d.layersValid.Load() {
		return d.layersCache, nil
	}

	layers, err := d.topoLayersLocked()
	if err != nil {
		return nil, err
	}

	// 更新缓存
	d.layersCache = layers
	d.layersValid.Store(true)

	return layers, nil
}

// topoLayersLocked 在已持有锁的情况下计算分层拓扑序
func (d *DAG) topoLayersLocked() ([][]ID, error) {
	inDegree := make(map[ID]int, len(d.nodes))
	for id := range d.nodes {
		inDegree[id] = 0
	}
	for _, n := range d.nodes {
		for to := range n.Successors {
			inDegree[to]++
		}
	}
	var layers [][]ID
	var q []ID
	for id, deg := range inDegree {
		if deg == 0 {
			q = append(q, id)
		}
	}
	for len(q) > 0 {
		curLevel := q
		layers = append(layers, curLevel)
		q = nil
		for _, id := range curLevel {
			for to := range d.nodes[id].Successors {
				inDegree[to]--
				if inDegree[to] == 0 {
					q = append(q, to)
				}
			}
		}
	}
	return layers, nil
}

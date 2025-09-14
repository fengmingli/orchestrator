package dag

import (
	"fmt"
	"strings"
	"sync"
)

// DAG 整个图
type DAG struct {
	nodes map[ID]*Node
	mu    sync.RWMutex
}

// NewDAG 从描述构建，返回环的具体路径（方便前端高亮）
func NewDAG(desc []Desc) (*DAG, error) {
	d := &DAG{nodes: make(map[ID]*Node, len(desc))}
	// 1. 建节点
	for _, n := range desc {
		d.nodes[n.ID] = &Node{
			ID:           n.ID,
			Mode:         n.Mode,
			Runner:       n.Runner,
			Successors:   make(map[ID]*Node),
			Predecessors: make(map[ID]*Node),
		}
	}
	// 2. 再建边（依赖 = 父 → 子）
	for _, n := range desc {
		to := d.nodes[n.ID] // 自己（子）
		for _, fromID := range n.Dependencies {
			from, ok := d.nodes[fromID]
			if !ok {
				return nil, fmt.Errorf("dependency %s not found", fromID)
			}
			// 正确方向：父 → 子
			from.Successors[n.ID] = to
			to.Predecessors[fromID] = from
		}
	}

	// 3. 环检测（Kahn）
	if cycle := d.findCycle(); cycle != nil {
		return nil, fmt.Errorf("cycle detected: %v", cycle)
	}
	return d, nil
}

// Kahn 算法返回环的节点序列（若无环返回 nil）
func (d *DAG) findCycle() []ID {
	inDegree := make(map[ID]int, len(d.nodes))
	for id := range d.nodes {
		inDegree[id] = 0
	}
	for _, n := range d.nodes {
		for to := range n.Successors {
			inDegree[to]++
		}
	}
	q := []ID{}
	for id, deg := range inDegree {
		if deg == 0 {
			q = append(q, id)
		}
	}
	cnt := 0
	order := make([]ID, 0, len(d.nodes))
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
	if cnt == len(d.nodes) {
		return nil
	}
	// 有环，找出环（DFS）
	visited := make(map[ID]bool)
	onStack := make(map[ID]bool)
	var cycle []ID
	var dfs func(ID) bool
	dfs = func(id ID) bool {
		visited[id] = true
		onStack[id] = true
		cycle = append(cycle, id)
		for to := range d.nodes[id].Successors {
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

// AddNode 运行时加节点（必须无环）
func (d *DAG) AddNode(n *Node) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.nodes[n.ID]; ok {
		return fmt.Errorf("node %s exists", n.ID)
	}
	d.nodes[n.ID] = n
	if cycle := d.findCycleLocked(); cycle != nil {
		delete(d.nodes, n.ID)
		return fmt.Errorf("add node introduces cycle: %v", cycle)
	}
	return nil
}

// AddEdge 加边（双向维护）
func (d *DAG) AddEdge(from, to ID) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	f, ok1 := d.nodes[from]
	t, ok2 := d.nodes[to]
	if !ok1 || !ok2 {
		return fmt.Errorf("node not found")
	}
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
	return nil
}

// findCycleLocked 在已持有 d.mu 的情况下检测环
// 返回构成环的节点 ID 序列（方便前端高亮），无环返回 nil
func (d *DAG) findCycleLocked() []ID {
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
	// Kahn 求拓扑序
	q := make([]ID, 0, len(d.nodes))
	for id, deg := range inDegree {
		if deg == 0 {
			q = append(q, id)
		}
	}
	cnt := 0
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		cnt++
		for to := range d.nodes[cur].Successors {
			inDegree[to]--
			if inDegree[to] == 0 {
				q = append(q, to)
			}
		}
	}
	if cnt == len(d.nodes) {
		return nil
	}
	// 存在环，用 DFS 找出具体环
	visited := make(map[ID]bool, len(d.nodes))
	onStack := make(map[ID]bool, len(d.nodes))
	var cycle []ID
	var dfs func(ID) bool
	dfs = func(id ID) bool {
		visited[id] = true
		onStack[id] = true
		cycle = append(cycle, id)
		for to := range d.nodes[id].Successors {
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

// ToGraphviz 导出 DOT
func (d *DAG) ToGraphviz() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	b := &strings.Builder{}
	b.WriteString("digraph G {\n")
	for _, n := range d.nodes {
		fmt.Fprintf(b, `"%s" [label="%s",shape=%s];`+"\n",
			n.ID, n.ID,
			map[RunMode]string{
				RunModeSerial:   "box",
				RunModeParallel: "ellipse",
				RunModeMixed:    "diamond",
			}[n.Mode])
		for to := range n.Successors {
			fmt.Fprintf(b, `"%s" -> "%s";`+"\n", n.ID, to)
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
			fmt.Fprintf(b, "%s-->;%s;\n", n.ID, to)
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

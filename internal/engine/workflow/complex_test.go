package workflow

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestComplexDAGExecution 测试复杂DAG的执行
func TestComplexDAGExecution(t *testing.T) {
	/*
		构造一个复杂的DAG:
		    A
		   / \
		  B   C
		 /|   |\
		D E   F G
		 \|   |/
		  H   I
		   \ /
		    J
	*/

	var execOrder []string
	var mu sync.Mutex

	recorder := func(name string) func() error {
		return func() error {
			time.Sleep(10 * time.Millisecond) // 模拟任务执行时间
			mu.Lock()
			execOrder = append(execOrder, name)
			mu.Unlock()
			return nil
		}
	}

	desc := []Desc{
		{ID: "A", Mode: RunModeSerial, Runner: recorder("A")},
		{ID: "B", Mode: RunModeParallel, Deps: []ID{"A"}, Runner: recorder("B")},
		{ID: "C", Mode: RunModeParallel, Deps: []ID{"A"}, Runner: recorder("C")},
		{ID: "D", Mode: RunModeParallel, Deps: []ID{"B"}, Runner: recorder("D")},
		{ID: "E", Mode: RunModeParallel, Deps: []ID{"B"}, Runner: recorder("E")},
		{ID: "F", Mode: RunModeParallel, Deps: []ID{"C"}, Runner: recorder("F")},
		{ID: "G", Mode: RunModeParallel, Deps: []ID{"C"}, Runner: recorder("G")},
		{ID: "H", Mode: RunModeSerial, Deps: []ID{"D", "E"}, Runner: recorder("H")},
		{ID: "I", Mode: RunModeSerial, Deps: []ID{"F", "G"}, Runner: recorder("I")},
		{ID: "J", Mode: RunModeSerial, Deps: []ID{"H", "I"}, Runner: recorder("J")},
	}

	dag, err := NewDAG(desc)
	assert.NoError(t, err)

	// 验证拓扑结构
	layers, err := dag.topoLayersCached()
	assert.NoError(t, err)
	t.Logf("拓扑分层: %v", layers)

	// 验证分层结构正确性
	assert.Equal(t, []ID{"A"}, layers[0])
	assert.ElementsMatch(t, []ID{"B", "C"}, layers[1])
	assert.ElementsMatch(t, []ID{"D", "E", "F", "G"}, layers[2])
	assert.ElementsMatch(t, []ID{"H", "I"}, layers[3])
	assert.Equal(t, []ID{"J"}, layers[4])

	scheduler := NewScheduler(dag, 8)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	err = scheduler.Run(ctx, nil)
	duration := time.Since(start)

	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	t.Logf("执行顺序: %v", execOrder)
	t.Logf("总执行时间: %v", duration)

	// 验证依赖关系
	aIndex := indexOf(execOrder, "A")
	bIndex := indexOf(execOrder, "B")
	cIndex := indexOf(execOrder, "C")
	dIndex := indexOf(execOrder, "D")
	eIndex := indexOf(execOrder, "E")
	fIndex := indexOf(execOrder, "F")
	gIndex := indexOf(execOrder, "G")
	hIndex := indexOf(execOrder, "H")
	iIndex := indexOf(execOrder, "I")
	jIndex := indexOf(execOrder, "J")

	// 验证依赖关系
	assert.True(t, aIndex < bIndex && aIndex < cIndex, "A应该在B、C之前")
	assert.True(t, bIndex < dIndex && bIndex < eIndex, "B应该在D、E之前")
	assert.True(t, cIndex < fIndex && cIndex < gIndex, "C应该在F、G之前")
	assert.True(t, dIndex < hIndex && eIndex < hIndex, "D、E应该在H之前")
	assert.True(t, fIndex < iIndex && gIndex < iIndex, "F、G应该在I之前")
	assert.True(t, hIndex < jIndex && iIndex < jIndex, "H、I应该在J之前")

	// 验证并行执行效率（理论上最短路径长度为5层，每层10ms，应该在100ms左右完成）
	assert.True(t, duration < 200*time.Millisecond, "并行执行应该在合理时间内完成")
}

// TestDAGResume 测试DAG恢复执行功能
func TestDAGResume(t *testing.T) {
	var execOrder []string
	var mu sync.Mutex

	recorder := func(name string) func() error {
		return func() error {
			mu.Lock()
			execOrder = append(execOrder, name)
			mu.Unlock()
			return nil
		}
	}

	desc := []Desc{
		{ID: "A", Mode: RunModeSerial, Runner: recorder("A")},
		{ID: "B", Mode: RunModeParallel, Deps: []ID{"A"}, Runner: recorder("B")},
		{ID: "C", Mode: RunModeParallel, Deps: []ID{"A"}, Runner: recorder("C")},
		{ID: "D", Mode: RunModeSerial, Deps: []ID{"B", "C"}, Runner: recorder("D")},
	}

	dag, err := NewDAG(desc)
	assert.NoError(t, err)

	scheduler := NewScheduler(dag, 4)

	// 模拟之前已经完成的节点
	resumeMap := map[ID]bool{
		"A": true,
		"B": true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = scheduler.Run(ctx, resumeMap)
	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	t.Logf("执行顺序: %v", execOrder)

	// 验证已完成的节点不会重新执行
	assert.NotContains(t, execOrder, "A", "已完成的A不应该重新执行")
	assert.NotContains(t, execOrder, "B", "已完成的B不应该重新执行")

	// 验证未完成的节点会执行
	assert.Contains(t, execOrder, "C", "未完成的C应该执行")
	assert.Contains(t, execOrder, "D", "未完成的D应该执行")

	// 验证执行顺序（C应该在D之前）
	cIndex := indexOf(execOrder, "C")
	dIndex := indexOf(execOrder, "D")
	assert.True(t, cIndex < dIndex, "C应该在D之前执行")
}

// TestDAGSnapshot 测试DAG快照功能
func TestDAGSnapshot(t *testing.T) {
	desc := []Desc{
		{ID: "A", Mode: RunModeSerial},
		{ID: "B", Mode: RunModeParallel, Deps: []ID{"A"}},
		{ID: "C", Mode: RunModeParallel, Deps: []ID{"A"}},
	}

	dag, err := NewDAG(desc)
	assert.NoError(t, err)

	// 测试结构快照
	structSnapshot := dag.SSnapshot()
	assert.Len(t, structSnapshot.Nodes, 3)

	// 验证节点信息
	nodeMap := make(map[ID]NodeSnapshot)
	for _, node := range structSnapshot.Nodes {
		nodeMap[node.ID] = node
	}

	assert.Equal(t, RunModeSerial, nodeMap["A"].Mode)
	assert.Empty(t, nodeMap["A"].Deps)

	assert.Equal(t, RunModeParallel, nodeMap["B"].Mode)
	assert.Equal(t, []ID{"A"}, nodeMap["B"].Deps)

	assert.Equal(t, RunModeParallel, nodeMap["C"].Mode)
	assert.Equal(t, []ID{"A"}, nodeMap["C"].Deps)

	// 测试MD5版本号生成
	md5Version, err := dag.GetStructuralMD5()
	assert.NoError(t, err)
	assert.NotEmpty(t, md5Version)
	t.Logf("DAG结构MD5: %s", md5Version)

	// 测试状态快照
	stateSnapshot := dag.Snapshot()
	assert.Len(t, stateSnapshot, 3)

	// 所有节点初始状态应该是Pending
	for id, state := range stateSnapshot {
		assert.Equal(t, StatePending, state, "节点%s的初始状态应该是Pending", id)
	}
}

// TestDAGVisualization 测试DAG可视化功能
func TestDAGVisualization(t *testing.T) {
	desc := []Desc{
		{ID: "Start", Mode: RunModeSerial},
		{ID: "Process1", Mode: RunModeParallel, Deps: []ID{"Start"}},
		{ID: "Process2", Mode: RunModeParallel, Deps: []ID{"Start"}},
		{ID: "End", Mode: RunModeMixed, Deps: []ID{"Process1", "Process2"}},
	}

	dag, err := NewDAG(desc)
	assert.NoError(t, err)

	// 测试Graphviz导出
	dot := dag.ToGraphviz()
	assert.Contains(t, dot, "digraph G")
	assert.Contains(t, dot, "Start")
	assert.Contains(t, dot, "Process1")
	assert.Contains(t, dot, "Process2")
	assert.Contains(t, dot, "End")
	assert.Contains(t, dot, "Start -> Process1")
	assert.Contains(t, dot, "Start -> Process2")
	assert.Contains(t, dot, "Process1 -> End")
	assert.Contains(t, dot, "Process2 -> End")

	t.Logf("Graphviz DOT:\n%s", dot)

	// 测试Mermaid导出
	mermaid := dag.ToMermaid()
	assert.Contains(t, mermaid, "graph TD")
	assert.Contains(t, mermaid, "Start --> Process1")
	assert.Contains(t, mermaid, "Start --> Process2")
	assert.Contains(t, mermaid, "Process1 --> End")
	assert.Contains(t, mermaid, "Process2 --> End")

	t.Logf("Mermaid图:\n%s", mermaid)
}

// TestLargeDAGPerformance 测试大型DAG的性能
func TestLargeDAGPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过大型DAG性能测试")
	}

	// 创建一个大型DAG（1000个节点）
	const nodeCount = 1000
	desc := make([]Desc, nodeCount)

	counter := func(i int) func() error {
		return func() error {
			// 模拟轻量级计算
			time.Sleep(time.Microsecond)
			return nil
		}
	}

	// 创建多层依赖结构
	for i := 0; i < nodeCount; i++ {
		nodeID := fmt.Sprintf("Node%04d", i)
		desc[i] = Desc{
			ID:     nodeID,
			Mode:   RunModeParallel,
			Runner: counter(i),
		}

		// 每个节点依赖前面几个节点（创建扇形依赖）
		if i > 0 {
			depCount := min(i, 3) // 最多依赖前3个节点
			deps := make([]ID, depCount)
			for j := 0; j < depCount; j++ {
				deps[j] = fmt.Sprintf("Node%04d", i-j-1)
			}
			desc[i].Deps = deps
		}
	}

	start := time.Now()
	dag, err := NewDAG(desc)
	buildTime := time.Since(start)

	assert.NoError(t, err)
	t.Logf("构建%d个节点的DAG耗时: %v", nodeCount, buildTime)

	// 测试拓扑排序性能
	start = time.Now()
	topo, err := dag.TopoSortCached()
	topoTime := time.Since(start)

	assert.NoError(t, err)
	assert.Len(t, topo, nodeCount)
	t.Logf("拓扑排序耗时: %v", topoTime)

	// 测试分层计算性能
	start = time.Now()
	layers, err := dag.topoLayersCached()
	layersTime := time.Since(start)

	assert.NoError(t, err)
	t.Logf("分层计算耗时: %v", layersTime)
	t.Logf("总共%d层", len(layers))

	// 测试执行性能
	scheduler := NewScheduler(dag, 50) // 使用50个并发工作者
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start = time.Now()
	err = scheduler.Run(ctx, nil)
	execTime := time.Since(start)

	assert.NoError(t, err)
	t.Logf("执行%d个节点耗时: %v", nodeCount, execTime)

	// 性能断言
	assert.True(t, buildTime < 1*time.Second, "构建大型DAG应在1秒内完成")
	assert.True(t, topoTime < 100*time.Millisecond, "拓扑排序应在100ms内完成")
	assert.True(t, layersTime < 100*time.Millisecond, "分层计算应在100ms内完成")
	assert.True(t, execTime < 10*time.Second, "执行应在10秒内完成")
}

// min 返回两个整数的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

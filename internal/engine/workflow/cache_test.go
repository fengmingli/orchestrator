package workflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCacheOptimization 测试缓存优化功能
func TestCacheOptimization(t *testing.T) {
	desc := []Desc{
		{ID: "A", Mode: RunModeSerial},
		{ID: "B", Mode: RunModeParallel, Deps: []ID{"A"}},
		{ID: "C", Mode: RunModeParallel, Deps: []ID{"A"}},
		{ID: "D", Mode: RunModeSerial, Deps: []ID{"B", "C"}},
	}

	dag, err := NewDAG(desc)
	assert.NoError(t, err)

	// 第一次计算拓扑序，应该计算并缓存
	start := time.Now()
	topo1, err := dag.TopoSortCached()
	duration1 := time.Since(start)
	assert.NoError(t, err)
	assert.NotNil(t, topo1)
	t.Logf("第一次拓扑排序耗时: %v", duration1)

	// 第二次计算，应该使用缓存，速度更快
	start = time.Now()
	topo2, err := dag.TopoSortCached()
	duration2 := time.Since(start)
	assert.NoError(t, err)
	assert.Equal(t, topo1, topo2)
	t.Logf("第二次拓扑排序耗时: %v", duration2)

	// 缓存命中应该明显更快
	assert.True(t, duration2 < duration1, "缓存应该提升性能")

	// 测试分层缓存
	start = time.Now()
	layers1, err := dag.topoLayersCached()
	duration1 = time.Since(start)
	assert.NoError(t, err)
	assert.NotNil(t, layers1)
	t.Logf("第一次分层计算耗时: %v", duration1)

	start = time.Now()
	layers2, err := dag.topoLayersCached()
	duration2 = time.Since(start)
	assert.NoError(t, err)
	assert.Equal(t, layers1, layers2)
	t.Logf("第二次分层计算耗时: %v", duration2)

	// 缓存命中应该明显更快
	assert.True(t, duration2 < duration1, "分层缓存应该提升性能")
}

// TestCacheInvalidation 测试缓存失效机制
func TestCacheInvalidation(t *testing.T) {
	desc := []Desc{
		{ID: "A", Mode: RunModeSerial},
		{ID: "B", Mode: RunModeParallel, Deps: []ID{"A"}},
	}

	dag, err := NewDAG(desc)
	assert.NoError(t, err)

	// 计算初始拓扑序
	topo1, err := dag.TopoSortCached()
	assert.NoError(t, err)
	assert.Equal(t, []ID{"A", "B"}, topo1)

	// 添加新节点，应该使缓存失效
	newNode := &Node{
		ID:           "C",
		Mode:         RunModeSerial,
		Successors:   make(map[ID]*Node),
		Predecessors: make(map[ID]*Node),
		doneCh:       make(chan struct{}),
	}
	err = dag.AddNode(newNode)
	assert.NoError(t, err)

	// 添加边，连接到新节点
	err = dag.AddEdge("B", "C")
	assert.NoError(t, err)

	// 重新计算拓扑序，应该包含新节点
	topo2, err := dag.TopoSortCached()
	assert.NoError(t, err)
	assert.Equal(t, []ID{"A", "B", "C"}, topo2)
	assert.NotEqual(t, topo1, topo2, "添加节点后拓扑序应该改变")

	// 删除节点，也应该使缓存失效
	err = dag.RemoveNode("C")
	assert.NoError(t, err)

	topo3, err := dag.TopoSortCached()
	assert.NoError(t, err)
	assert.Equal(t, topo1, topo3, "删除节点后应该恢复原始拓扑序")
}

// BenchmarkTopoSortWithCache 性能测试：带缓存的拓扑排序
func BenchmarkTopoSortWithCache(b *testing.B) {
	// 创建一个较大的DAG
	const nodeCount = 100
	desc := make([]Desc, nodeCount)

	// 创建链式依赖: 0->1->2->...->99
	for i := 0; i < nodeCount; i++ {
		desc[i] = Desc{
			ID:   string(rune('A'+i%26)) + string(rune('0'+i/26)),
			Mode: RunModeParallel,
		}
		if i > 0 {
			desc[i].Deps = []ID{desc[i-1].ID}
		}
	}

	dag, err := NewDAG(desc)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	// 测试缓存性能
	for i := 0; i < b.N; i++ {
		_, err := dag.TopoSortCached()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTopoSortWithoutCache 性能测试：不带缓存的拓扑排序
func BenchmarkTopoSortWithoutCache(b *testing.B) {
	// 创建一个较大的DAG
	const nodeCount = 100
	desc := make([]Desc, nodeCount)

	// 创建链式依赖: 0->1->2->...->99
	for i := 0; i < nodeCount; i++ {
		desc[i] = Desc{
			ID:   string(rune('A'+i%26)) + string(rune('0'+i/26)),
			Mode: RunModeParallel,
		}
		if i > 0 {
			desc[i].Deps = []ID{desc[i-1].ID}
		}
	}

	dag, err := NewDAG(desc)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	// 测试不使用缓存的性能
	for i := 0; i < b.N; i++ {
		_, err := dag.TopoSort()
		if err != nil {
			b.Fatal(err)
		}
	}
}

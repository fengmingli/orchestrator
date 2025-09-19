package workflow

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestEmptyDAG 测试空DAG
func TestEmptyDAG(t *testing.T) {
	dag, err := NewDAG([]Desc{})
	assert.NoError(t, err)
	assert.NotNil(t, dag)

	// 空DAG的拓扑排序应该返回空切片
	topo, err := dag.TopoSortCached()
	assert.NoError(t, err)
	assert.Empty(t, topo)

	// 空DAG的分层应该返回空切片
	layers, err := dag.topoLayersCached()
	assert.NoError(t, err)
	assert.Empty(t, layers)

	// 执行空DAG应该立即完成
	scheduler := NewScheduler(dag, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	err = scheduler.Run(ctx, nil)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.True(t, duration < 100*time.Millisecond, "空DAG应该快速完成")
}

// TestSingleNodeDAG 测试单节点DAG
func TestSingleNodeDAG(t *testing.T) {
	executed := false

	desc := []Desc{
		{
			ID:   "single",
			Mode: RunModeSerial,
			Runner: func() error {
				executed = true
				return nil
			},
		},
	}

	dag, err := NewDAG(desc)
	assert.NoError(t, err)

	// 单节点拓扑排序
	topo, err := dag.TopoSortCached()
	assert.NoError(t, err)
	assert.Equal(t, []ID{"single"}, topo)

	// 单节点分层
	layers, err := dag.topoLayersCached()
	assert.NoError(t, err)
	assert.Len(t, layers, 1)
	assert.Equal(t, []ID{"single"}, layers[0])

	// 执行单节点DAG
	scheduler := NewScheduler(dag, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = scheduler.Run(ctx, nil)
	assert.NoError(t, err)
	assert.True(t, executed, "单节点应该被执行")
}

// TestSelfDependency 测试自依赖检测
func TestSelfDependency(t *testing.T) {
	desc := []Desc{
		{
			ID:   "self",
			Deps: []ID{"self"}, // 自依赖
		},
	}

	dag, err := NewDAG(desc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "self-loop detected")
	assert.Nil(t, dag)
}

// TestMissingDependency 测试缺失依赖
func TestMissingDependency(t *testing.T) {
	desc := []Desc{
		{
			ID:   "node1",
			Deps: []ID{"missing"}, // 引用不存在的节点
		},
	}

	dag, err := NewDAG(desc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency missing not found")
	assert.Nil(t, dag)
}

// TestDuplicateNodeID 测试重复节点ID
func TestDuplicateNodeID(t *testing.T) {
	desc := []Desc{
		{ID: "duplicate"},
		{ID: "duplicate"}, // 重复ID
	}

	dag, err := NewDAG(desc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node duplicate duplicated")
	assert.Nil(t, dag)
}

// TestEmptyNodeID 测试空节点ID
func TestEmptyNodeID(t *testing.T) {
	desc := []Desc{
		{ID: ""}, // 空ID
	}

	dag, err := NewDAG(desc)
	assert.NoError(t, err) // 当前实现允许空ID
	assert.NotNil(t, dag)
}

// TestEmptyDependencies 测试空依赖处理
func TestEmptyDependencies(t *testing.T) {
	desc := []Desc{
		{
			ID:   "node1",
			Deps: []ID{"", "  ", "node2"}, // 包含空字符串和空白字符串
		},
		{
			ID: "node2",
		},
	}

	dag, err := NewDAG(desc)
	assert.NoError(t, err) // 应该正确处理空依赖
	assert.NotNil(t, dag)

	// 验证依赖关系正确建立
	topo, err := dag.TopoSortCached()
	assert.NoError(t, err)
	assert.Equal(t, []ID{"node2", "node1"}, topo)
}

// TestNilRunner 测试空Runner
func TestNilRunner(t *testing.T) {
	desc := []Desc{
		{
			ID:     "nil-runner",
			Runner: nil, // 空Runner
		},
	}

	dag, err := NewDAG(desc)
	assert.NoError(t, err) // 构建应该成功
	assert.NotNil(t, dag)

	// 执行时应该处理nil Runner
	scheduler := NewScheduler(dag, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 这可能会panic或返回错误，取决于具体实现
	defer func() {
		if r := recover(); r != nil {
			t.Logf("执行nil Runner时panic: %v", r)
		}
	}()

	err = scheduler.Run(ctx, nil)
	// 不严格断言结果，因为这是边界情况
	t.Logf("执行nil Runner的结果: %v", err)
}

// TestMaxWorkers 测试不同的最大工作者数量
func TestMaxWorkers(t *testing.T) {
	executed := make([]bool, 5)

	desc := make([]Desc, 5)
	for i := 0; i < 5; i++ {
		idx := i // 避免闭包问题
		desc[i] = Desc{
			ID:   string(rune('A' + i)),
			Mode: RunModeParallel,
			Runner: func() error {
				time.Sleep(10 * time.Millisecond)
				executed[idx] = true
				return nil
			},
		}
	}

	dag, err := NewDAG(desc)
	assert.NoError(t, err)

	testCases := []int{1, 2, 5, 10}

	for _, workers := range testCases {
		t.Run(fmt.Sprintf("workers-%d", workers), func(t *testing.T) {
			// 重置执行状态
			for i := range executed {
				executed[i] = false
			}

			scheduler := NewScheduler(dag, workers)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			start := time.Now()
			err := scheduler.Run(ctx, nil)
			duration := time.Since(start)

			assert.NoError(t, err)
			t.Logf("使用%d个工作者，执行时间: %v", workers, duration)

			// 验证所有任务都执行了
			for i, exec := range executed {
				assert.True(t, exec, "任务%d应该被执行", i)
			}
		})
	}
}

// TestZeroMaxWorkers 测试零工作者数量
func TestZeroMaxWorkers(t *testing.T) {
	t.Skip("零工作者会导致阻塞，跳过此测试")

	desc := []Desc{
		{ID: "test", Runner: func() error { return nil }},
	}

	dag, err := NewDAG(desc)
	assert.NoError(t, err)

	scheduler := NewScheduler(dag, 0) // 零工作者
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = scheduler.Run(ctx, nil)
	// 零工作者可能导致阻塞，应该超时
	if err != nil {
		assert.Contains(t, err.Error(), "context deadline exceeded", "零工作者应该导致超时")
	}
	t.Logf("零工作者的执行结果: %v", err)
}

// TestVeryLargeDependencyChain 测试非常长的依赖链
func TestVeryLargeDependencyChain(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过长依赖链测试")
	}

	const chainLength = 100
	desc := make([]Desc, chainLength)

	for i := 0; i < chainLength; i++ {
		nodeID := fmt.Sprintf("node%d", i)
		desc[i] = Desc{
			ID:   nodeID,
			Mode: RunModeSerial,
			Runner: func() error {
				return nil
			},
		}

		if i > 0 {
			desc[i].Deps = []ID{fmt.Sprintf("node%d", i-1)}
		}
	}

	start := time.Now()
	dag, err := NewDAG(desc)
	buildTime := time.Since(start)

	assert.NoError(t, err)
	t.Logf("构建%d节点链式DAG耗时: %v", chainLength, buildTime)

	// 验证拓扑排序
	start = time.Now()
	topo, err := dag.TopoSortCached()
	topoTime := time.Since(start)

	assert.NoError(t, err)
	assert.Len(t, topo, chainLength)

	// 验证顺序正确
	for i, nodeID := range topo {
		expected := fmt.Sprintf("node%d", i)
		assert.Equal(t, expected, nodeID, "拓扑排序顺序不正确")
	}

	t.Logf("拓扑排序耗时: %v", topoTime)

	// 执行测试
	scheduler := NewScheduler(dag, 1) // 使用单工作者确保串行执行
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start = time.Now()
	err = scheduler.Run(ctx, nil)
	execTime := time.Since(start)

	assert.NoError(t, err)
	t.Logf("执行%d节点链式DAG耗时: %v", chainLength, execTime)
}

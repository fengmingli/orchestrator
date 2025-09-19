package workflow

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestSelfLoop 测试自环检测
func TestSelfLoop(t *testing.T) {
	tests := []struct {
		name string
		desc []Desc
	}{
		{
			name: "直接自环",
			desc: []Desc{
				{ID: "A", Deps: []ID{"A"}}, // A依赖自己
			},
		},
		{
			name: "多节点中的自环",
			desc: []Desc{
				{ID: "A", Deps: []ID{}},
				{ID: "B", Deps: []ID{"A", "B"}}, // B依赖A和自己
				{ID: "C", Deps: []ID{"B"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dag, err := NewDAG(tt.desc)
			assert.Error(t, err, "应该检测到自环")
			assert.Nil(t, dag, "自环时不应该创建DAG")
			assert.Contains(t, err.Error(), "self-loop detected", "错误信息应该包含自环提示")
			t.Logf("自环检测错误: %v", err)
		})
	}
}

// TestSimpleCycles 测试简单环检测
func TestSimpleCycles(t *testing.T) {
	tests := []struct {
		name string
		desc []Desc
	}{
		{
			name: "两节点环: A->B->A",
			desc: []Desc{
				{ID: "A", Deps: []ID{"B"}},
				{ID: "B", Deps: []ID{"A"}},
			},
		},
		{
			name: "三节点环: A->B->C->A",
			desc: []Desc{
				{ID: "A", Deps: []ID{"C"}},
				{ID: "B", Deps: []ID{"A"}},
				{ID: "C", Deps: []ID{"B"}},
			},
		},
		{
			name: "四节点环: A->B->C->D->A",
			desc: []Desc{
				{ID: "A", Deps: []ID{"D"}},
				{ID: "B", Deps: []ID{"A"}},
				{ID: "C", Deps: []ID{"B"}},
				{ID: "D", Deps: []ID{"C"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dag, err := NewDAG(tt.desc)
			assert.Error(t, err, "应该检测到环")
			assert.Nil(t, dag, "存在环时不应该创建DAG")
			assert.Contains(t, err.Error(), "cycle detected", "错误信息应该包含环检测提示")
			t.Logf("环检测错误: %v", err)
		})
	}
}

// TestComplexCycles 测试复杂环检测
func TestComplexCycles(t *testing.T) {
	tests := []struct {
		name string
		desc []Desc
	}{
		{
			name: "多个入口的环",
			desc: []Desc{
				{ID: "A", Deps: []ID{}},         // 无依赖
				{ID: "B", Deps: []ID{"A"}},      // B依赖A
				{ID: "C", Deps: []ID{"B", "E"}}, // C依赖B和E（形成环）
				{ID: "D", Deps: []ID{"C"}},      // D依赖C
				{ID: "E", Deps: []ID{"D"}},      // E依赖D（形成环：C->D->E->C）
			},
		},
		{
			name: "钻石形状中的环",
			desc: []Desc{
				{ID: "A", Deps: []ID{}},         // 根节点
				{ID: "B", Deps: []ID{"A"}},      // B依赖A
				{ID: "C", Deps: []ID{"A"}},      // C依赖A
				{ID: "D", Deps: []ID{"B", "C"}}, // D依赖B和C
				{ID: "E", Deps: []ID{"D", "A"}}, // E依赖D和A（正常）
				{ID: "F", Deps: []ID{"E"}},      // F依赖E
				{ID: "G", Deps: []ID{"F", "B"}}, // G依赖F和B
				{ID: "H", Deps: []ID{"G"}},      // H依赖G
				{ID: "I", Deps: []ID{"H", "D"}}, // I依赖H和D
				{ID: "B", Deps: []ID{"I"}},      // 重新定义B依赖I（会被后面的覆盖，但这里测试重复ID）
			},
		},
		{
			name: "嵌套环",
			desc: []Desc{
				{ID: "A", Deps: []ID{}},
				{ID: "B", Deps: []ID{"A"}},
				{ID: "C", Deps: []ID{"F"}},      // C依赖F
				{ID: "D", Deps: []ID{"C"}},      // D依赖C
				{ID: "E", Deps: []ID{"D"}},      // E依赖D
				{ID: "F", Deps: []ID{"E", "B"}}, // F依赖E和B（形成环：C->D->E->F->C）
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dag, err := NewDAG(tt.desc)
			assert.Error(t, err, "应该检测到环")
			assert.Nil(t, dag, "存在环时不应该创建DAG")

			// 错误信息应该包含环检测或重复节点提示
			errorMsg := err.Error()
			hasCycleError := len(errorMsg) > 0 && (strings.Contains(errorMsg, "cycle detected") ||
				strings.Contains(errorMsg, "duplicated"))

			assert.True(t, hasCycleError, "应该检测到环或重复节点错误: %s", errorMsg)
			t.Logf("检测错误: %v", err)
		})
	}
}

// TestNoCycle 测试无环的正常情况
func TestNoCycle(t *testing.T) {
	tests := []struct {
		name string
		desc []Desc
	}{
		{
			name: "线性依赖",
			desc: []Desc{
				{ID: "A", Deps: []ID{}},
				{ID: "B", Deps: []ID{"A"}},
				{ID: "C", Deps: []ID{"B"}},
				{ID: "D", Deps: []ID{"C"}},
			},
		},
		{
			name: "钻石形状无环",
			desc: []Desc{
				{ID: "A", Deps: []ID{}},
				{ID: "B", Deps: []ID{"A"}},
				{ID: "C", Deps: []ID{"A"}},
				{ID: "D", Deps: []ID{"B", "C"}},
			},
		},
		{
			name: "复杂树状结构",
			desc: []Desc{
				{ID: "Root", Deps: []ID{}},
				{ID: "L1A", Deps: []ID{"Root"}},
				{ID: "L1B", Deps: []ID{"Root"}},
				{ID: "L2A", Deps: []ID{"L1A"}},
				{ID: "L2B", Deps: []ID{"L1A"}},
				{ID: "L2C", Deps: []ID{"L1B"}},
				{ID: "L3A", Deps: []ID{"L2A", "L2B"}},
				{ID: "L3B", Deps: []ID{"L2C"}},
				{ID: "Final", Deps: []ID{"L3A", "L3B"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dag, err := NewDAG(tt.desc)
			assert.NoError(t, err, "无环的DAG应该创建成功")
			assert.NotNil(t, dag, "应该成功创建DAG")

			// 验证拓扑排序能够成功
			topo, err := dag.TopoSortCached()
			assert.NoError(t, err, "无环DAG应该能够进行拓扑排序")
			assert.Len(t, topo, len(tt.desc), "拓扑排序应该包含所有节点")
			t.Logf("拓扑排序结果: %v", topo)
		})
	}
}

// TestCycleDetectionAfterAddNode 测试动态添加节点时的环检测
func TestCycleDetectionAfterAddNode(t *testing.T) {
	// 创建一个初始无环的DAG
	desc := []Desc{
		{ID: "A", Deps: []ID{}},
		{ID: "B", Deps: []ID{"A"}},
		{ID: "C", Deps: []ID{"B"}},
	}

	dag, err := NewDAG(desc)
	assert.NoError(t, err)
	assert.NotNil(t, dag)

	// 添加一个会形成环的节点
	newNode := &Node{
		ID:           "D",
		Mode:         RunModeSerial,
		Successors:   make(map[ID]*Node),
		Predecessors: make(map[ID]*Node),
		doneCh:       make(chan struct{}),
	}

	err = dag.AddNode(newNode)
	assert.NoError(t, err, "添加节点本身应该成功")

	// 添加会形成环的边：A -> D (这样就形成了 A -> B -> C, A -> D，无环)
	err = dag.AddEdge("A", "D")
	assert.NoError(t, err, "添加正常边应该成功")

	// 现在添加会形成环的边：D -> A (这样就形成了 A -> D -> A 的环)
	err = dag.AddEdge("D", "A")
	assert.Error(t, err, "添加形成环的边应该失败")
	assert.Contains(t, err.Error(), "cycle", "错误信息应该包含环检测提示")
	t.Logf("动态环检测错误: %v", err)
}

// TestCycleDetectionAfterAddEdge 测试动态添加边时的环检测
func TestCycleDetectionAfterAddEdge(t *testing.T) {
	// 创建一个初始无环的DAG
	desc := []Desc{
		{ID: "A", Deps: []ID{}},
		{ID: "B", Deps: []ID{"A"}},
		{ID: "C", Deps: []ID{"B"}},
		{ID: "D", Deps: []ID{}}, // D是独立的节点
	}

	dag, err := NewDAG(desc)
	assert.NoError(t, err)
	assert.NotNil(t, dag)

	// 验证初始状态无环
	topo, err := dag.TopoSortCached()
	assert.NoError(t, err)
	t.Logf("初始拓扑排序: %v", topo)

	// 添加正常的边
	err = dag.AddEdge("C", "D")
	assert.NoError(t, err, "添加正常边应该成功")

	// 现在尝试添加会形成环的边
	err = dag.AddEdge("D", "A") // 这会形成环：A -> B -> C -> D -> A
	assert.Error(t, err, "添加形成环的边应该失败")
	assert.Contains(t, err.Error(), "cycle", "错误信息应该包含环检测提示")
	t.Logf("边添加环检测错误: %v", err)

	// 验证DAG仍然是无环的（添加失败的边不应该影响DAG状态）
	topo2, err := dag.TopoSortCached()
	assert.NoError(t, err, "DAG应该仍然无环")
	t.Logf("添加边失败后的拓扑排序: %v", topo2)
}

// TestComplexCyclePatterns 测试各种复杂的环模式
func TestComplexCyclePatterns(t *testing.T) {
	tests := []struct {
		name        string
		desc        []Desc
		expectCycle bool
	}{
		{
			name: "蝴蝶结形状-无环",
			desc: []Desc{
				{ID: "A", Deps: []ID{}},
				{ID: "B", Deps: []ID{"A"}},
				{ID: "C", Deps: []ID{"A"}},
				{ID: "D", Deps: []ID{"B", "C"}}, // 汇聚点
				{ID: "E", Deps: []ID{"D"}},
				{ID: "F", Deps: []ID{"D"}},
				{ID: "G", Deps: []ID{"E", "F"}},
			},
			expectCycle: false,
		},
		{
			name: "多入口环",
			desc: []Desc{
				{ID: "A", Deps: []ID{}},
				{ID: "B", Deps: []ID{}},         // 另一个入口
				{ID: "C", Deps: []ID{"A", "B"}}, // C依赖A和B
				{ID: "D", Deps: []ID{"C"}},
				{ID: "E", Deps: []ID{"D"}},
				{ID: "A", Deps: []ID{"E"}}, // 重新定义A依赖E，形成环
			},
			expectCycle: true,
		},
		{
			name: "星形结构-无环",
			desc: []Desc{
				{ID: "Center", Deps: []ID{}},
				{ID: "N1", Deps: []ID{"Center"}},
				{ID: "N2", Deps: []ID{"Center"}},
				{ID: "N3", Deps: []ID{"Center"}},
				{ID: "N4", Deps: []ID{"Center"}},
				{ID: "N5", Deps: []ID{"Center"}},
			},
			expectCycle: false,
		},
		{
			name: "图形状环",
			desc: []Desc{
				{ID: "A", Deps: []ID{}},
				{ID: "B", Deps: []ID{"A"}},
				{ID: "C", Deps: []ID{"A"}},
				{ID: "D", Deps: []ID{"B"}},
				{ID: "E", Deps: []ID{"C"}},
				{ID: "F", Deps: []ID{"D", "E"}}, // F依赖D和E
				{ID: "G", Deps: []ID{"F"}},
				{ID: "H", Deps: []ID{"G"}},
				{ID: "B", Deps: []ID{"H"}}, // 重定义B依赖H，形成长环
			},
			expectCycle: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dag, err := NewDAG(tt.desc)

			if tt.expectCycle {
				assert.Error(t, err, "应该检测到环")
				assert.Nil(t, dag, "存在环时不应该创建DAG")
				if err != nil {
					// 错误可能是环检测或重复节点ID
					errorMsg := err.Error()
					hasCycleError := strings.Contains(errorMsg, "cycle") ||
						strings.Contains(errorMsg, "duplicated")
					assert.True(t, hasCycleError, "应该包含环或重复节点错误信息: %s", errorMsg)
				}
			} else {
				assert.NoError(t, err, "无环的DAG应该创建成功")
				assert.NotNil(t, dag, "应该成功创建DAG")

				// 验证能够进行拓扑排序
				topo, err := dag.TopoSortCached()
				assert.NoError(t, err, "应该能够进行拓扑排序")
				assert.Len(t, topo, len(tt.desc), "拓扑排序应该包含所有节点")
			}

			t.Logf("测试结果: 预期环=%v, 实际错误=%v", tt.expectCycle, err)
		})
	}
}

// TestCycleDetectionPerformance 测试环检测的性能
func TestCycleDetectionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过环检测性能测试")
	}

	// 创建一个大的无环DAG
	const nodeCount = 1000
	desc := make([]Desc, nodeCount)

	for i := 0; i < nodeCount; i++ {
		nodeID := fmt.Sprintf("Node%04d", i)
		desc[i] = Desc{
			ID:   nodeID,
			Mode: RunModeParallel,
		}

		// 每个节点依赖前面几个节点（创建复杂依赖，但无环）
		if i > 0 {
			depCount := min(i, 5) // 最多依赖前5个节点
			deps := make([]ID, depCount)
			for j := 0; j < depCount; j++ {
				deps[j] = fmt.Sprintf("Node%04d", i-j-1)
			}
			desc[i].Deps = deps
		}
	}

	// 测试无环检测性能
	start := time.Now()
	dag, err := NewDAG(desc)
	duration := time.Since(start)

	assert.NoError(t, err, "大型无环DAG应该创建成功")
	assert.NotNil(t, dag)
	t.Logf("大型DAG(%d节点)环检测耗时: %v", nodeCount, duration)

	// 现在添加一个会形成环的边来测试环检测
	start = time.Now()
	err = dag.AddEdge("Node0999", "Node0000") // 让最后一个节点依赖第一个节点
	duration = time.Since(start)

	assert.Error(t, err, "应该检测到环")
	assert.Contains(t, err.Error(), "cycle", "应该包含环检测错误")
	t.Logf("大型DAG环检测耗时: %v", duration)

	// 环检测应该在合理时间内完成
	assert.True(t, duration < 1*time.Second, "环检测应该在1秒内完成")
}

package dag

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestSerialParallelMixed 验证串并混合执行顺序：A(串) -> [B||C||D](并) -> E(串)
func TestSerialParallelMixed(t *testing.T) {
	// 1. 每次测试独享记录切片
	var (
		order []string
		mu    sync.Mutex
	)

	// 2. 工厂函数：避免闭包捕获循环变量
	makeRecorder := func(name string) func() error {
		return func() error {
			// 关键：先记录，再耗时
			mu.Lock()
			order = append(order, name)
			mu.Unlock()

			time.Sleep(10 * time.Millisecond) // 模拟任务耗时
			return nil
		}
	}

	// 3. 构造 DAG 描述
	desc := []Desc{
		{ID: "A", Mode: RunModeSerial, Runner: makeRecorder("A")},
		{ID: "B", Mode: RunModeParallel, Dependencies: []string{"A"}, Runner: makeRecorder("B")},
		{ID: "C", Mode: RunModeParallel, Dependencies: []string{"A"}, Runner: makeRecorder("C")},
		{ID: "D", Mode: RunModeParallel, Dependencies: []string{"A"}, Runner: makeRecorder("D")},
		{ID: "E", Mode: RunModeSerial, Dependencies: []string{"B", "C", "D"}, Runner: makeRecorder("E")},
	}

	// 4. 建图 & 调度
	d, err := NewDAG(desc)
	layers, _ := d.topoLayers()
	t.Logf("拓扑分层: %#v", layers)

	assert.NoError(t, err)
	sched := NewScheduler(d, 10)
	err = sched.Run(context.Background(), nil)
	assert.NoError(t, err)

	// 5. 断言执行顺序
	mu.Lock()
	defer mu.Unlock()
	t.Logf("实际执行顺序: %v", order)
	assert.Equal(t, "A", order[0], "A 必须第一个执行")
	assert.Equal(t, "E", order[len(order)-1], "E 必须最后一个执行")
}

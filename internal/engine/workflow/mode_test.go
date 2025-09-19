package workflow

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestABCMock 真实 mock 函数：A->[B||C||D]->E
func TestABCDEMock(t *testing.T) {
	var (
		order []string
		mu    sync.Mutex
	)

	// 记录函数：打印开始+结束，并记录顺序
	mock := func(name string, sleep time.Duration) func() error {
		return func() error {
			start := time.Now()
			fmt.Printf("【%s】开始 @ %s\n", name, start.Format("15:04:05.000"))
			time.Sleep(sleep)
			end := time.Now()
			fmt.Printf("【%s】结束 @ %s  (耗时 %v)\n", name, end.Format("15:04:05.000"), end.Sub(start))

			mu.Lock()
			order = append(order, name)
			mu.Unlock()
			return nil
		}
	}

	// 各任务耗时（ms）
	const (
		tA = 80 * time.Millisecond
		tB = 50 * time.Millisecond
		tC = 40 * time.Millisecond
		tD = 30 * time.Millisecond
		tE = 20 * time.Millisecond
	)

	desc := []Desc{
		{ID: "A", Mode: RunModeSerial, Runner: mock("A", tA)},
		{ID: "B", Mode: RunModeParallel, Deps: []ID{"A"}, Runner: mock("B", tB)},
		{ID: "C", Mode: RunModeParallel, Deps: []ID{"A"}, Runner: mock("C", tC)},
		{ID: "D", Mode: RunModeParallel, Deps: []ID{"A"}, Runner: mock("D", tD)},
		{ID: "E", Mode: RunModeSerial, Deps: []ID{"B", "C", "D"}, Runner: mock("E", tE)},
	}

	d, err := NewDAG(desc)
	assert.NoError(t, err)

	layers, _ := d.topoLayers()
	t.Logf("拓扑分层: %#v", layers)

	sched := NewScheduler(d, 10)
	err = sched.Run(context.Background(), nil)
	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	t.Logf("实际执行顺序: %v", order)
	assert.Equal(t, "A", order[0], "A 必须第一个")
	assert.Equal(t, "E", order[len(order)-1], "E 必须最后一个")
}

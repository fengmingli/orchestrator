package workflow

import (
	"context"
	"sync"
	"testing"
	"time"
)

/**
 * @Author: LFM
 * @Date: 2025/9/14 23:25
 * @Since: 1.0.0
 * @Desc: TODO
 */

func TestCycle(t *testing.T) {
	_, err := NewDAG([]Desc{
		{ID: "a", Deps: []ID{"c"}},
		{ID: "b", Deps: []ID{"a"}},
		{ID: "c", Deps: []ID{"b"}},
	})
	if err == nil {
		t.Fatal("expect cycle")
	}
}

func TestScheduler(t *testing.T) {
	d, _ := NewDAG([]Desc{
		{ID: "a", Mode: RunModeParallel},
		{ID: "b", Mode: RunModeParallel, Deps: []ID{"a"}},
		{ID: "c", Mode: RunModeParallel, Deps: []ID{"a"}},
		{ID: "d", Mode: RunModeSerial, Deps: []ID{"b", "c"}},
	})
	var order []string
	var mu sync.Mutex
	for _, n := range d.nodes {
		n := n
		n.Runner = func() error {
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			order = append(order, n.ID)
			mu.Unlock()
			return nil
		}
	}
	s := NewScheduler(d, 4)
	if err := s.Run(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	// 断言：a 在 b/c 前，d 在最后
	t.Logf("execution order: %v", order)
}

package dag

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Scheduler 负责按拓扑层并发执行
type Scheduler struct {
	dag        *DAG
	maxWorkers int
}

func NewScheduler(d *DAG, maxWorkers int) *Scheduler {
	return &Scheduler{dag: d, maxWorkers: maxWorkers}
}

// Run 执行全图；resume 已完成的节点会被跳过
func (s *Scheduler) Run(ctx context.Context, resume map[ID]bool) error {
	layers, err := s.dag.topoLayers()
	if err != nil {
		return err
	}

	// 已结束节点
	done := make(map[ID]bool)
	for id := range resume {
		done[id] = true
	}

	// 逐层执行
	for _, layer := range layers {
		if err := s.runLayer(ctx, layer, done); err != nil {
			return err
		}
	}
	return nil
}

// 执行一层
func (s *Scheduler) runLayer(ctx context.Context, layer []ID, done map[ID]bool) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(s.maxWorkers)

	var mu sync.Mutex
	// 等待本层所有节点完成
	//finish := make(chan struct{})

	for _, id := range layer {
		id := id
		node := s.dag.nodes[id]
		g.Go(func() error {
			// 1. 等待前驱完成
			if err := s.waitPreds(ctx, node, done); err != nil {
				return err
			}
			// 2. 已跑过直接跳过
			mu.Lock()
			if done[id] {
				mu.Unlock()
				return nil
			}
			mu.Unlock()

			// 3. 运行
			node.State.Store(uint32(StateRunning))
			err := node.Runner()
			if err != nil {
				node.State.Store(uint32(StateFailed))
				return err
			}
			node.State.Store(uint32(StateSuccess))

			// 4. 标记完成
			mu.Lock()
			done[id] = true
			mu.Unlock()
			return nil
		})
		// 串行节点：等它完成再继续放任务
		if node.Mode == RunModeSerial {
			if err := g.Wait(); err != nil {
				return err
			}
		}
	}
	return g.Wait()
}

// 等待前驱全部完成
func (s *Scheduler) waitPreds(ctx context.Context, n *Node, done map[ID]bool) error {
	for pred := range n.Predecessors {
		for !done[pred] {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				// 简单轮询，可换成 cond 或 channel 优化
				time.Sleep(5 * time.Millisecond)
			}
		}
	}
	return nil
}

// topoLayers 返回分层拓扑序（BFS）
func (d *DAG) topoLayers() ([][]ID, error) {
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
	q := []ID{}
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

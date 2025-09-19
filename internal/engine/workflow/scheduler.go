package workflow

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Scheduler 负责按拓扑层并发执行
type Scheduler struct {
	dag          *DAG
	maxWorkers   int
	cancel       context.CancelFunc // 全局取消（用于 Abort）
	mu           sync.Mutex
	hasReportErr bool // 是否出现 SkipButReport 的失败
}

func NewScheduler(d *DAG, maxWorkers int) *Scheduler {
	return &Scheduler{dag: d, maxWorkers: maxWorkers}
}

// WithCancel 注入全局 cancel，用于 Abort 策略时取消还在执行的节点
func (s *Scheduler) WithCancel(cancel context.CancelFunc) *Scheduler {
	s.cancel = cancel
	return s
}

// Run 执行全图；resume 已完成的节点会被跳过
func (s *Scheduler) Run(ctx context.Context, resume map[ID]bool) error {
	layers, err := s.dag.topoLayersCached()
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
		if err = s.runLayer(ctx, layer, done); err != nil {
			return err
		}
	}
	s.mu.Lock()
	report := s.hasReportErr
	s.mu.Unlock()
	if report {
		return fmt.Errorf("dag executed with skipped failures")
	}
	return nil
}

// 执行一层
func (s *Scheduler) runLayer(ctx context.Context, layer []ID, done map[ID]bool) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(s.maxWorkers)
	var mu sync.Mutex
	// 等待本层所有节点完成
	for _, id := range layer {
		idTmp := id
		node := s.dag.nodes[idTmp]
		g.Go(func() error {
			// 1. 等待前驱完成
			if err := s.waitPreds(ctx, node, done); err != nil {
				return err
			}
			// 2. 已跑过直接跳过
			mu.Lock()
			if done[idTmp] {
				mu.Unlock()
				//resume 内存直接表示 "成功完成"
				node.State.Store(uint32(StateSucceeded))
				return nil
			}
			mu.Unlock()
			// 3. 运行
			node.State.Store(uint32(StateRunning))
			var err error
			if node.Runner != nil {
				err = node.Runner() // 业务逻辑
			}
			// 如果Runner为nil，视为成功完成
			if err != nil {
				// 根据策略处理失败
				node.State.Store(uint32(StateFailed))
				switch node.policy.OnFailure {
				case FailureAbort:
					// 标记全局失败并取消
					s.dag.failed.Store(true)
					if s.cancel != nil {
						s.cancel()
					}
					// 仍需标记完成，唤醒后继等待者
					mu.Lock()
					done[idTmp] = true
					mu.Unlock()
					node.MarkDone()
					// 返回错误以终止 errgroup
					return err
				case FailureSkip:
					// 标记完成但不影响整体，后继会将此视为可继续
					// 记录为 failed，但返回 nil，让其他任务继续
					mu.Lock()
					done[idTmp] = true
					mu.Unlock()
					node.MarkDone()
					return nil
				case FailureSkipButReport:
					// 继续执行，但在 Run() 结束时返回整体错误
					s.mu.Lock()
					s.hasReportErr = true
					s.mu.Unlock()
					mu.Lock()
					done[idTmp] = true
					mu.Unlock()
					node.MarkDone()
					return nil
				default:
					// 默认 Abort
					s.dag.failed.Store(true)
					if s.cancel != nil {
						s.cancel()
					}
					mu.Lock()
					done[idTmp] = true
					mu.Unlock()
					node.MarkDone()
					return err
				}
			}
			// 成功
			node.State.Store(uint32(StateSucceeded))
			// 4. 标记完成
			mu.Lock()
			done[idTmp] = true
			mu.Unlock()
			// 使用 once 确保只 close 一次，即使被多个 goroutine 调用也安全
			node.MarkDone()
			// 返回错误
			return err
		})
		// 串行节点：等它完成再继续放任务
		if node.Mode == RunModeSerial {
			if err := g.Wait(); err != nil {
				return err
			}
		}
	}
	if err := g.Wait(); err != nil {
		return err
	}
	// 本层无致命错误，允许继续。但若存在 SkipButReport，最后统一返回错误
	return nil
}

// 等待前驱全部完成
func (s *Scheduler) waitPreds(ctx context.Context, n *Node, done map[ID]bool) error {
	for _, predNode := range n.Predecessors {
		if predNode == nil || predNode.doneCh == nil {
			continue
		}
		//如果前驱还没有完成，监听他的doneCh
		if !done[predNode.ID] {
			select {
			case <-predNode.doneCh: //读操作：等待 channel 关闭
				// 前驱完成后，如果失败，根据其策略决定是否阻断
				if predNode.State.Load() == uint32(StateFailed) {
					switch predNode.policy.OnFailure {
					case FailureAbort:
						// 视为致命失败，阻断当前节点
						return fmt.Errorf("predecessor %s failed (abort)", predNode.ID)
					case FailureSkip, FailureSkipButReport:
						// 允许继续
					default:
						return fmt.Errorf("predecessor %s failed", predNode.ID)
					}
				}
				continue
			case <-ctx.Done():
				return ctx.Err()
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

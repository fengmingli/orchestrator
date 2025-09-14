package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/fengmingli/orchestrator/internal/dal"
	dag2 "github.com/fengmingli/orchestrator/internal/engine/dag"
	"github.com/fengmingli/orchestrator/internal/model"
	"github.com/fengmingli/orchestrator/pkg/logger"
	"github.com/fengmingli/orchestrator/pkg/retry"
	"golang.org/x/sync/errgroup"
)

type Executor struct {
	lg logger.Logger
}

func NewExecutor() *Executor {
	return &Executor{lg: logger.L().WithFields(logger.Field{
		Key: "component", Value: "executor",
	})}
}

func (e *Executor) Run(ctx context.Context, executionID string) error {
	// 1. 加载数据
	var exec model.Execution
	if err := dal.DB.First(&exec, "id = ?", executionID).Error; err != nil {
		return err
	}
	var tpl model.Template
	if err := dal.DB.Preload("Steps").First(&tpl, "id = ?", exec.TemplateID).Error; err != nil {
		return err
	}
	dag, err := dag2.NewDAG(tpl.Steps)
	if err != nil {
		return err
	}
	topo, err := dag.Topological()
	if err != nil {
		return err
	}

	// 2. 初始化 stepExecution 记录
	step2record := make(map[string]*model.StepExecution)
	for _, s := range tpl.Steps {
		rec := &model.StepExecution{
			ID:          fmt.Sprintf("%s_%s", executionID, s.StepKey),
			ExecutionID: executionID,
			StepID:      s.ID,
			Status:      "pending",
		}
		if err := dal.DB.Create(rec).Error; err != nil {
			return err
		}
		step2record[s.StepKey] = rec
	}

	// 3. 更新 execution 状态
	now := time.Now()
	exec.Status = "running"
	exec.StartedAt = &now
	dal.DB.Save(&exec)

	// 4. 并发调度
	var mu sync.Mutex
	markFinished := func(key string, output, errStr string, status string) {
		mu.Lock()
		defer mu.Unlock()
		rec := step2record[key]
		rec.Output = output
		rec.Error = errStr
		rec.Status = status
		rec.FinishedAt = ptrTime(time.Now())
		dal.DB.Save(rec)
	}

	g, ctx := errgroup.WithContext(ctx)
	done := make(map[string]bool)
	var doneMu sync.Mutex
	waitCond := sync.NewCond(&doneMu)

	for _, key := range topo {
		key := key
		s := stepByKey(tpl.Steps, key)
		g.Go(func() error {
			// 等待依赖完成
			var deps []string
			json.Unmarshal(s.Dependencies, &deps)
			doneMu.Lock()
			for !allDone(done, deps) {
				waitCond.Wait()
			}
			doneMu.Unlock()

			// 执行
			rec := step2record[key]
			rec.Status = "running"
			rec.StartedAt = ptrTime(time.Now())
			dal.DB.Save(rec)

			output, err := e.runStepWithRetry(ctx, s, executionID)
			if err != nil {
				markFinished(key, output, err.Error(), "failed")
				return err
			}
			markFinished(key, output, "", "success")

			doneMu.Lock()
			done[key] = true
			waitCond.Broadcast()
			doneMu.Unlock()
			return nil
		})
		if s.Mode == "serial" {
			_ = g.Wait() // 串行等待
		}
	}

	err = g.Wait()
	finish := time.Now()
	if err != nil {
		exec.Status = "failed"
	} else {
		exec.Status = "success"
	}
	exec.FinishedAt = &finish
	dal.DB.Save(&exec)
	return err
}

func (e *Executor) runStepWithRetry(ctx context.Context, step model.TemplateStep, executionID string) (string, error) {
	runner, ok := GetRunner(step.Type)
	if !ok {
		return "", fmt.Errorf("unknown step type %s", step.Type)
	}
	var output string
	var err error
	retryFn := func() error {
		c, cancel := context.WithTimeout(ctx, time.Duration(step.TimeoutSec)*time.Second)
		defer cancel()
		output, err = runner.Run(c, step, executionID)
		return err
	}
	retryCfg := retry.Config{Times: step.RetryTimes, Delay: 1 * time.Second}
	err = retry.Do(retryCfg, retryFn)
	return output, err
}

// 工具
func stepByKey(steps []model.TemplateStep, key string) *model.TemplateStep {
	for i := range steps {
		if steps[i].StepKey == key {
			return &steps[i]
		}
	}
	return nil
}
func allDone(done map[string]bool, deps []string) bool {
	for _, d := range deps {
		if !done[d] {
			return false
		}
	}
	return true
}
func ptrTime(t time.Time) *time.Time { return &t }

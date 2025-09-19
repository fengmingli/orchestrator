package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fengmingli/orchestrator/internal/dal"
	"github.com/fengmingli/orchestrator/internal/engine/workflow"
	"github.com/fengmingli/orchestrator/internal/model"
	"github.com/fengmingli/orchestrator/pkg/retry"
)

// Executor: DAG 调度器封装，符合 doc.txt 流程
// 1) 构建 DAG（Definition）
// 2) 使用 Scheduler 分层调度并发执行
// 3) 每个节点通过可重试执行器包装（DB/Log/Metrics Hook）
// 4) 等待节点完成，触发下游
// 5) 汇总结果：更新 Execution 成功/失败
type Executor struct {
	maxWorkers int
}

func NewExecutor() *Executor {
	return &Executor{maxWorkers: 16}
}

func (e *Executor) Run(ctx context.Context, executionID string) error {
	// 1. 加载 Execution 与 Template
	var exec model.Execution
	if err := dal.DB.First(&exec, "id = ?", executionID).Error; err != nil {
		return err
	}
	var tpl model.Template
	if err := dal.DB.Preload("Steps").First(&tpl, "id = ?", exec.TemplateID).Error; err != nil {
		return err
	}

	// 2. 初始化步骤执行记录（pending）
	stepRec := make(map[string]*model.StepExecution, len(tpl.Steps))
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
		stepRec[s.StepKey] = rec
	}

	// 3. 标记 Execution 运行中
	now := time.Now()
	exec.Status = "running"
	exec.StartedAt = &now
	_ = dal.DB.Save(&exec).Error

	// 4. 组装 DAG 描述（每个节点包装成可重试 Runner）
	descs := make([]workflow.Desc, 0, len(tpl.Steps))
	for _, s := range tpl.Steps {
		// 解析依赖
		var deps []string
		_ = json.Unmarshal([]byte(s.Dependencies), &deps)
		// 运行模式
		mode := workflow.AsRunMode(s.Mode)
		// 包装 runner：附带超时/重试和 DB Hook
		runFn := e.wrapRunner(ctx, s, executionID, stepRec[s.StepKey])
		descs = append(descs, workflow.Desc{
			ID:     s.StepKey,
			Mode:   mode,
			Deps:   deps,
			Runner: runFn,
			Policy: workflow.ExecutionPolicy{OnFailure: parseFailureAction(s.OnFailure)},
		})
	}

	// 5. 构建 DAG + 调度执行
	dag, err := workflow.NewDAG(descs)
	if err != nil {
		// DAG 构建失败，直接标记 Execution 失败
		exec.Status = "failed"
		finish := time.Now()
		exec.FinishedAt = &finish
		_ = dal.DB.Save(&exec).Error
		return err
	}
	// 创建可取消的执行上下文，便于 Abort 时快速失败
	execCtx, execCancel := context.WithCancel(ctx)
	defer execCancel()
	sched := workflow.NewScheduler(dag, e.maxWorkers).WithCancel(execCancel)
	err = sched.Run(execCtx, nil)

	// 6. 汇总 Execution 状态
	finish := time.Now()
	if err != nil {
		exec.Status = "failed"
	} else {
		exec.Status = "success"
	}
	exec.FinishedAt = &finish
	_ = dal.DB.Save(&exec).Error
	return err
}

// wrapRunner 将 TemplateStep 转为 workflow.Runner：
// - 更新 StepExecution 状态（running/success/failed）
// - 应用超时与重试策略
func (e *Executor) wrapRunner(parentCtx context.Context, step model.TemplateStep, executionID string, rec *model.StepExecution) func() error {
	return func() error {
		// 标记 running
		rec.Status = "running"
		rec.StartedAt = ptrTime(time.Now())
		_ = dal.DB.Save(rec).Error

		// 查找实际 StepRunner
		runner, ok := GetRunner(step.Type)
		if !ok {
			err := fmt.Errorf("unknown step type %s", step.Type)
			// 立即失败
			rec.Error = err.Error()
			rec.Status = "failed"
			rec.FinishedAt = ptrTime(time.Now())
			_ = dal.DB.Save(rec).Error
			return err
		}

		var output string
		var finalErr error

		// 执行函数（带超时）
		call := func() error {
			// 每次重试都创建新的超时上下文
			to := time.Duration(step.TimeoutSec) * time.Second
			if to <= 0 {
				to = 30 * time.Second
			}
			ctx, cancel := context.WithTimeout(parentCtx, to)
			defer cancel()
			var err error
			output, err = runner.Run(ctx, step, executionID)
			return err
		}

		// 重试
		rcfg := retry.Config{Times: step.RetryTimes, Delay: 1 * time.Second}
		finalErr = retry.Do(rcfg, call)

		// 回写结果
		rec.Output = output
		if finalErr != nil {
			rec.Error = finalErr.Error()
			rec.Status = "failed"
		} else {
			rec.Status = "success"
		}
		rec.FinishedAt = ptrTime(time.Now())
		_ = dal.DB.Save(rec).Error
		return finalErr
	}
}

// 小工具
func ptrTime(t time.Time) *time.Time { return &t }

func parseFailureAction(s string) workflow.FailureAction {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "skip":
		return workflow.FailureSkip
	case "skip_but_report", "skip-but-report", "skipbutreport":
		return workflow.FailureSkipButReport
	case "abort", "":
		fallthrough
	default:
		return workflow.FailureAbort
	}
}

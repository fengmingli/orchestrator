package executorx

import (
	"context"
	"fmt"

	"github.com/fengmingli/orchestrator/internal/engine/task"
)

// WarpExecRequest 包装的执行请求
type WarpExecRequest struct {
	Task    task.Task
	Ctx     context.Context
	OrderID string
}

// WarpExecResult 包装的执行结果
type WarpExecResult struct {
	task.ExecResult
	ExecStatus    task.ExecStatus
	FailureReason string
	Err           error
}

// WrapTask 将task包装成workflow可用的runner函数
func WrapTask(ctx context.Context, taskInst task.Task, executor *RetryableExecutor, taskID string) func() error {
	return func() error {
		warpExecRequest := WarpExecRequest{
			Task:    taskInst,
			Ctx:     ctx,
			OrderID: taskID,
		}
		warpExecResult := executor.Execute(warpExecRequest)

		// 检查执行状态，如果失败则返回错误
		if warpExecResult.ExecStatus == task.ExecStatusWithFailed {
			if warpExecResult.Err != nil {
				return warpExecResult.Err
			}
			// 如果没有具体错误，创建一个通用错误
			return fmt.Errorf("任务执行失败: %s", warpExecResult.FailureReason)
		}

		return warpExecResult.Err
	}
}

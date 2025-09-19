package executorx

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fengmingli/
	"github.com/fengmingli/orchestrator/internal/engine/task"
	"github.com/pkg/errors"
	"github.com/cenkalti/backoff/v4"
)

// Executor 定义了执行任务的接口。
type Executor interface {
	Execute(request WarpExecRequest) WarpExecResult
}

// --- Hook 机制 ---
// Hook 定义了任务生命周期中可以插入的钩子函数。
type Hook interface {
	OnRunning(ctx context.Context, taskInst task.Task, result WarpExecResult)
	OnSuccess(ctx context.Context, taskInst task.Task, result WarpExecResult)
	OnFailure(ctx context.Context, taskInst task.Task, result WarpExecResult)
}

// --- 核心执行器 ---
// RetryableExecutor 是一个支持重试、超时和 Hook 的执行器。
type RetryableExecutor struct {
	logger        *logrus.Entry
	maxRetries    uint
	retryDelay    time.Duration
	maxDelay      time.Duration
	timeout       time.Duration
	backoffFactor float64
	retryOn       func(err error) bool
	hook          Hook
	mu            sync.RWMutex // 保护配置hooks []Hook
}

// NewRetryableExecutor 创建一个新的 RetryableExecutor 实例。
func NewRetryableExecutor() *RetryableExecutor {
	return &RetryableExecutor{
		maxRetries:    10, // 默认重试10次
		retryDelay:    500 * time.Millisecond,
		maxDelay:      1 * time.Second,
		timeout:       10 * time.Minute,
		backoffFactor: 2,
		logger:        logrus.NewEntry(logrus.New()),
	}
}

// WithLogger 设置日志记录器
func (re *RetryableExecutor) WithLogger(logger *logrus.Entry) *RetryableExecutor {
	if logger != nil {
		re.logger = logger
	}
	return re
}

// WithTimeout 设置执行超时时间
func (re *RetryableExecutor) WithTimeout(timeout time.Duration) *RetryableExecutor {
	if timeout > 0 {
		re.timeout = timeout
	}
	return re
}

// WithBackoffFactor 设置退避因子
func (re *RetryableExecutor) WithBackoffFactor(factor float64) *RetryableExecutor {
	if factor > 1 {
		re.backoffFactor = factor
	}
	return re
}

// WithMaxDelay 设置最大延迟
func (re *RetryableExecutor) WithMaxDelay(delay time.Duration) *RetryableExecutor {
	if delay > 0 {
		re.maxDelay = delay
	}
	return re
}

// WithMaxRetries 设置最大重试次数。
func (re *RetryableExecutor) WithMaxRetries(maxRetries uint) *RetryableExecutor {
	if maxRetries >= 0 {
		re.maxRetries = maxRetries
	}
	return re
}

// WithRetryDelay 设置重试间隔。
func (re *RetryableExecutor) WithRetryDelay(retryDelay time.Duration) *RetryableExecutor {
	if retryDelay > 0 {
		re.retryDelay = retryDelay
	}
	return re
}

// WithHook 注入数据交互钩子
func (re *RetryableExecutor) WithHook(hook Hook) *RetryableExecutor {
	re.mu.Lock()
	defer re.mu.Unlock()
	if hook != nil {
		re.hook = hook
	}
	return re
}

// Execute 实现 Executor 接口，执行给定的任务。
func (re *RetryableExecutor) Execute(req WarpExecRequest) WarpExecResult {
	ctx := req.Ctx
	taskInst := req.Task
	var finalWarpExecResult WarpExecResult
	// 1. 参数校验
	if err := taskInst.Validate(); err != nil {
		finalWarpExecResult.ExecStatus = task.ExecStatusWithFailed
		finalWarpExecResult.FailureReason = fmt.Sprintf("参数校验失败，原因:%v", err.Error())
		if re.hook != nil {
			re.hook.OnFailure(ctx, taskInst, finalWarpExecResult)
		}
		return finalWarpExecResult
	}
	stepStartTime := time.Now()
	// 执行步骤入口（核心）
	execResultCh := make(chan WarpExecResult, 1)
	go func() {
		defer close(execResultCh) // 确保通道被关闭
		execResultCh <- re.doWithRetry(ctx, taskInst)
	}()
	select {
	case result := <-execResultCh:
		if result.Err != nil {
			re.logger.Errorf("任务单:%s 步骤ID:%s 步骤名称:%s 执行错误: %v", req.OrderID, taskInst.GetID(), taskInst.GetName(), result.Err)
			finalWarpExecResult.ExecStatus = task.ExecStatusWithFailed
			finalWarpExecResult.FailureReason = fmt.Sprintf("执行失败，原因:%v", result.Err.Error())
			finalWarpExecResult.Err = result.Err // 设置错误字段
			if re.hook != nil {
				re.hook.OnFailure(ctx, taskInst, finalWarpExecResult)
			}
			return finalWarpExecResult
		}
		finalWarpExecResult = result
	case <-ctx.Done():
		re.logger.Errorf("任务单:%s，步骤ID:%s 步骤名称:%s 执行超时: %v", req.OrderID, taskInst.GetID(), taskInst.GetName(), ctx.Err())
		finalWarpExecResult.ExecStatus = task.ExecStatusWithFailed
		finalWarpExecResult.FailureReason = "超时，将快速失败"
		if re.hook != nil {
			re.hook.OnFailure(ctx, taskInst, finalWarpExecResult)
		}
		return finalWarpExecResult
	}
	//执行业务逻辑成功后
	finalWarpExecResult.ExecStatus = task.ExecStatusWithSuccess
	if re.hook != nil {
		re.hook.OnSuccess(ctx, taskInst, finalWarpExecResult)
	}
	stepEndTime := time.Now()
		re.logger.Infof("任务单:%s 步骤:%s 执行完成，开始时间：%s 结束时间：%s 耗时: %v",
			req.OrderID, taskInst.GetName(), stepStartTime.Format("15:04:05.000"),
			req.OrderID, taskInst.GetName(), stepStartTime.Format("15:04:05.000"), 
			stepEndTime.Format("15:04:05.000"), stepEndTime.Sub(stepStartTime))
	}
	return finalWarpExecResult
}
func (re *RetryableExecutor) doWithRetry(ctx context.Context, taskInst task.Task) WarpExecResult {
	var warpExecResult WarpExecResult

	
	// 配置指数退避
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = re.retryDelay
	b.MaxInterval = re.maxDelay
	b.Multiplier = re.backoffFactor

	
	// 限制重试次数

	
	operation := func() error {
		start := time.Now()
		execResult, callErr := taskInst.Execute(ctx)

		
		if callErr != nil {
			retryCount++
			if re.logger != nil {
				re.logger.Errorf("重试中(%d/%d) 任务ID:%s 执行:%s 执行失败:%v", retryCount, re.maxRetries, taskInst.GetID(), taskInst.GetName(), callErr.Error())
			}
			return callErr

		
		// 成功时填充结果
		warpExecResult.ExecResult = execResult
		warpExecResult.ExecResult.Duration = duration
		warpExecResult.ExecResult.StartTime = start
		warpExecResult.ExecResult.FinishTime = time.Now()
		return nil

	
	retryErr := backoff.Retry(operation, backoffWithMaxRetries)
	if retryErr != nil {
		warpExecResult.Err = errors.Wrapf(retryErr, "任务ID:%s 执行:%s 失败", taskInst.GetID(), taskInst.GetName())
		return warpExecResult
	}
	return warpExecResult
}

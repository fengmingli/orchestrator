package executorx

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fengmingli/orchestrator/internal/engine/task"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTask 模拟任务
type MockTask struct {
	task.BaseTask
	executeFunc func(ctx context.Context) (task.ExecResult, error)
}

func (m *MockTask) Execute(ctx context.Context) (task.ExecResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx)
	}
	return task.ExecResult{
		TaskID:    m.ID,
		Output:    "mock result",
		StartTime: time.Now(),
	}, nil
}

// MockHook 模拟Hook
type MockHook struct {
	mock.Mock
}

func (m *MockHook) OnRunning(ctx context.Context, taskInst task.Task, result WarpExecResult) {
	m.Called(ctx, taskInst, result)
}

func (m *MockHook) OnSuccess(ctx context.Context, taskInst task.Task, result WarpExecResult) {
	m.Called(ctx, taskInst, result)
}

func (m *MockHook) OnFailure(ctx context.Context, taskInst task.Task, result WarpExecResult) {
	m.Called(ctx, taskInst, result)
}

// TestNewRetryableExecutor 测试创建重试执行器
func TestNewRetryableExecutor(t *testing.T) {
	executor := NewRetryableExecutor()

	assert.NotNil(t, executor)
	assert.Equal(t, uint(10), executor.maxRetries)
	assert.Equal(t, 500*time.Millisecond, executor.retryDelay)
	assert.Equal(t, 1*time.Second, executor.maxDelay)
	assert.Equal(t, 10*time.Minute, executor.timeout)
	assert.Equal(t, float64(2), executor.backoffFactor)
}

// TestRetryableExecutorConfiguration 测试执行器配置
func TestRetryableExecutorConfiguration(t *testing.T) {
	executor := NewRetryableExecutor().
		WithMaxRetries(5).
		WithRetryDelay(1 * time.Second)

	assert.Equal(t, uint(5), executor.maxRetries)
	assert.Equal(t, 1*time.Second, executor.retryDelay)
}

// TestRetryableExecutorWithHook 测试Hook配置
func TestRetryableExecutorWithHook(t *testing.T) {
	hook := &MockHook{}
	executor := NewRetryableExecutor().WithHook(hook)

	assert.Equal(t, hook, executor.hook)
}

// TestExecuteSuccess 测试成功执行
func TestExecuteSuccess(t *testing.T) {
	// 创建模拟任务
	mockTask := &MockTask{
		BaseTask: task.BaseTask{
			ID:   "test-task",
			Name: "Test Task",
			Type: task.TaskTypeFunc,
		},
	}

	// 创建Mock Hook
	mockHook := &MockHook{}
	mockHook.On("OnSuccess", mock.Anything, mock.Anything, mock.Anything).Return()

	// 创建执行器
	executor := NewRetryableExecutor().
		WithHook(mockHook).
		WithMaxRetries(3)

	// 创建请求
	request := WarpExecRequest{
		Task:    mockTask,
		Ctx:     context.Background(),
		OrderID: "order-123",
	}

	// 执行
	result := executor.Execute(request)

	// 验证结果
	assert.Equal(t, task.ExecStatusWithSuccess, result.ExecStatus)
	assert.Nil(t, result.Err)
	assert.Equal(t, "mock result", result.Output)

	// 验证Hook被调用
	mockHook.AssertCalled(t, "OnSuccess", mock.Anything, mock.Anything, mock.Anything)
}

// TestExecuteValidationFailure 测试验证失败
func TestExecuteValidationFailure(t *testing.T) {
	// 创建无效任务（空ID）
	mockTask := &MockTask{
		BaseTask: task.BaseTask{
			ID:   "", // 空ID会导致验证失败
			Name: "Test Task",
			Type: task.TaskTypeFunc,
		},
	}

	mockHook := &MockHook{}
	mockHook.On("OnFailure", mock.Anything, mock.Anything, mock.Anything).Return()

	executor := NewRetryableExecutor().WithHook(mockHook)

	request := WarpExecRequest{
		Task:    mockTask,
		Ctx:     context.Background(),
		OrderID: "order-123",
	}

	result := executor.Execute(request)

	assert.Equal(t, task.ExecStatusWithFailed, result.ExecStatus)
	assert.Contains(t, result.FailureReason, "参数校验失败")
	mockHook.AssertCalled(t, "OnFailure", mock.Anything, mock.Anything, mock.Anything)
}

// TestExecuteTaskFailure 测试任务执行失败
func TestExecuteTaskFailure(t *testing.T) {
	// 创建会失败的任务
	mockTask := &MockTask{
		BaseTask: task.BaseTask{
			ID:   "test-task",
			Name: "Test Task",
			Type: task.TaskTypeFunc,
		},
		executeFunc: func(ctx context.Context) (task.ExecResult, error) {
			return task.ExecResult{}, errors.New("task execution failed")
		},
	}

	mockHook := &MockHook{}
	mockHook.On("OnFailure", mock.Anything, mock.Anything, mock.Anything).Return()

	executor := NewRetryableExecutor().
		WithHook(mockHook).
		WithMaxRetries(2)

	request := WarpExecRequest{
		Task:    mockTask,
		Ctx:     context.Background(),
		OrderID: "order-123",
	}

	result := executor.Execute(request)

	assert.Equal(t, task.ExecStatusWithFailed, result.ExecStatus)
	assert.Contains(t, result.FailureReason, "执行失败")
	assert.NotNil(t, result.Err)
	mockHook.AssertCalled(t, "OnFailure", mock.Anything, mock.Anything, mock.Anything)
}

// TestExecuteWithRetry 测试重试机制
func TestExecuteWithRetry(t *testing.T) {
	attemptCount := 0

	// 创建前两次失败，第三次成功的任务
	mockTask := &MockTask{
		BaseTask: task.BaseTask{
			ID:   "retry-task",
			Name: "Retry Task",
			Type: task.TaskTypeFunc,
		},
		executeFunc: func(ctx context.Context) (task.ExecResult, error) {
			attemptCount++
			if attemptCount < 3 {
				return task.ExecResult{}, errors.New("temporary failure")
			}
			return task.ExecResult{
				TaskID: "retry-task",
				Output: "success after retry",
			}, nil
		},
	}

	mockHook := &MockHook{}
	mockHook.On("OnSuccess", mock.Anything, mock.Anything, mock.Anything).Return()

	executor := NewRetryableExecutor().
		WithHook(mockHook).
		WithMaxRetries(5).
		WithRetryDelay(10 * time.Millisecond)

	request := WarpExecRequest{
		Task:    mockTask,
		Ctx:     context.Background(),
		OrderID: "order-123",
	}

	result := executor.Execute(request)

	assert.Equal(t, task.ExecStatusWithSuccess, result.ExecStatus)
	assert.Equal(t, "success after retry", result.Output)
	assert.Equal(t, 3, attemptCount) // 验证重试了3次
	mockHook.AssertCalled(t, "OnSuccess", mock.Anything, mock.Anything, mock.Anything)
}

// TestExecuteTimeout 测试执行超时
func TestExecuteTimeout(t *testing.T) {
	// 创建长时间运行的任务
	mockTask := &MockTask{
		BaseTask: task.BaseTask{
			ID:   "timeout-task",
			Name: "Timeout Task",
			Type: task.TaskTypeFunc,
		},
		executeFunc: func(ctx context.Context) (task.ExecResult, error) {
			select {
			case <-time.After(2 * time.Second):
				return task.ExecResult{}, nil
			case <-ctx.Done():
				return task.ExecResult{}, ctx.Err()
			}
		},
	}

	mockHook := &MockHook{}
	mockHook.On("OnFailure", mock.Anything, mock.Anything, mock.Anything).Return()

	executor := NewRetryableExecutor().WithHook(mockHook)

	// 创建会很快取消的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	request := WarpExecRequest{
		Task:    mockTask,
		Ctx:     ctx,
		OrderID: "order-123",
	}

	result := executor.Execute(request)

	assert.Equal(t, task.ExecStatusWithFailed, result.ExecStatus)
	assert.Contains(t, result.FailureReason, "超时")
	mockHook.AssertCalled(t, "OnFailure", mock.Anything, mock.Anything, mock.Anything)
}

// TestWrapTask 测试任务包装函数
func TestWrapTask(t *testing.T) {
	mockTask := &MockTask{
		BaseTask: task.BaseTask{
			ID:   "wrap-task",
			Name: "Wrap Task",
			Type: task.TaskTypeFunc,
		},
	}

	executor := NewRetryableExecutor()
	ctx := context.Background()

	// 包装任务
	wrappedFunc := WrapTask(ctx, mockTask, executor, "task-123")

	// 执行包装的函数
	err := wrappedFunc()

	assert.NoError(t, err)
}

// TestWrapTaskWithError 测试包装任务错误处理
func TestWrapTaskWithError(t *testing.T) {
	mockTask := &MockTask{
		BaseTask: task.BaseTask{
			ID:   "wrap-error-task",
			Name: "Wrap Error Task",
			Type: task.TaskTypeFunc,
		},
		executeFunc: func(ctx context.Context) (task.ExecResult, error) {
			return task.ExecResult{}, errors.New("execution failed")
		},
	}

	executor := NewRetryableExecutor()
	ctx := context.Background()

	// 包装任务
	wrappedFunc := WrapTask(ctx, mockTask, executor, "task-123")

	// 执行包装的函数
	err := wrappedFunc()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execution failed")
}

// TestExecutorWithLogger 测试带日志的执行器
func TestExecutorWithLogger(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	mockTask := &MockTask{
		BaseTask: task.BaseTask{
			ID:   "logged-task",
			Name: "Logged Task",
			Type: task.TaskTypeFunc,
		},
	}

	executor := NewRetryableExecutor()
	executor.logger = logger

	request := WarpExecRequest{
		Task:    mockTask,
		Ctx:     context.Background(),
		OrderID: "order-123",
	}

	result := executor.Execute(request)

	assert.Equal(t, task.ExecStatusWithSuccess, result.ExecStatus)
}

// TestExecutorWithoutHook 测试没有Hook的执行器
func TestExecutorWithoutHook(t *testing.T) {
	mockTask := &MockTask{
		BaseTask: task.BaseTask{
			ID:   "no-hook-task",
			Name: "No Hook Task",
			Type: task.TaskTypeFunc,
		},
	}

	executor := NewRetryableExecutor() // 没有设置Hook

	request := WarpExecRequest{
		Task:    mockTask,
		Ctx:     context.Background(),
		OrderID: "order-123",
	}

	// 应该不会panic
	result := executor.Execute(request)

	assert.Equal(t, task.ExecStatusWithSuccess, result.ExecStatus)
}

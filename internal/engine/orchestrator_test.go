package engine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fengmingli/orchestrator/internal/engine/executorx"
	"github.com/fengmingli/orchestrator/internal/engine/task"
	"github.com/fengmingli/orchestrator/internal/engine/workflow"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestTaskOrchestratorBasic 测试基本功能
func TestTaskOrchestratorBasic(t *testing.T) {
	orchestrator := NewTaskOrchestrator()

	assert.NotNil(t, orchestrator)
	assert.NotNil(t, orchestrator.executor)
	assert.NotNil(t, orchestrator.logger)
	assert.Equal(t, 10, orchestrator.maxWorkers)
}

// TestTaskOrchestratorConfiguration 测试配置
func TestTaskOrchestratorConfiguration(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	orchestrator := NewTaskOrchestrator().
		WithLogger(logger).
		WithMaxWorkers(20).
		WithRetryConfig(5, 2*time.Second)

	assert.Equal(t, logger, orchestrator.logger)
	assert.Equal(t, 20, orchestrator.maxWorkers)
}

// TestSimpleWorkflowExecution 测试简单工作流执行
func TestSimpleWorkflowExecution(t *testing.T) {
	orchestrator := NewTaskOrchestrator()

	// 创建简单的函数任务
	task1 := task.NewFuncTask("task1", "Task 1", func(ctx context.Context) (interface{}, error) {
		return "result1", nil
	})

	task2 := task.NewFuncTask("task2", "Task 2", func(ctx context.Context) (interface{}, error) {
		return "result2", nil
	})

	// 创建工作流定义
	definitions := []TaskDefinition{
		{
			ID:           "task1",
			Name:         "Task 1",
			Task:         task1,
			Dependencies: nil,
			Mode:         workflow.RunModeSerial,
			Policy:       workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Task:         task2,
			Dependencies: []string{"task1"}, // 依赖task1
			Mode:         workflow.RunModeSerial,
			Policy:       workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort},
		},
	}

	// 执行工作流
	ctx := context.Background()
	result, err := orchestrator.Execute(ctx, definitions)

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Empty(t, result.Error)
	assert.Len(t, result.TaskResults, 2)
	assert.NotZero(t, result.Duration)
}

// TestParallelWorkflowExecution 测试并行工作流执行
func TestParallelWorkflowExecution(t *testing.T) {
	orchestrator := NewTaskOrchestrator()

	// 创建多个并行任务
	tasks := make([]task.Task, 3)
	for i := 0; i < 3; i++ {
		taskID := fmt.Sprintf("parallel-task-%d", i+1)
		tasks[i] = task.NewFuncTask(taskID, fmt.Sprintf("Parallel Task %d", i+1),
			func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond) // 模拟执行时间
				return fmt.Sprintf("result-%d", i+1), nil
			})
	}

	// 创建并行工作流
	definitions, err := orchestrator.CreateParallelWorkflow(tasks...)
	assert.NoError(t, err)

	// 执行工作流
	ctx := context.Background()
	start := time.Now()
	result, err := orchestrator.Execute(ctx, definitions)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Len(t, result.TaskResults, 3)

	// 并行执行应该比串行快
	assert.True(t, duration < 250*time.Millisecond, "并行执行应该在250ms内完成")
}

// TestHTTPTaskIntegration 测试HTTP任务集成
func TestHTTPTaskIntegration(t *testing.T) {
	// 创建测试HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	orchestrator := NewTaskOrchestrator()

	// 创建HTTP任务
	httpTask := task.NewHTTPTask("http-task", "HTTP Task", "GET", server.URL)

	definitions := []TaskDefinition{
		{
			ID:           "http-task",
			Name:         "HTTP Task",
			Task:         httpTask,
			Dependencies: nil,
			Mode:         workflow.RunModeSerial,
			Policy:       workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort},
		},
	}

	// 执行
	ctx := context.Background()
	result, err := orchestrator.Execute(ctx, definitions)

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Len(t, result.TaskResults, 1)
}

// TestShellTaskIntegration 测试Shell任务集成
func TestShellTaskIntegration(t *testing.T) {
	orchestrator := NewTaskOrchestrator()

	// 创建Shell任务
	shellTask := task.NewShellTask("shell-task", "Shell Task", "echo 'Hello from shell'")

	definitions := []TaskDefinition{
		{
			ID:           "shell-task",
			Name:         "Shell Task",
			Task:         shellTask,
			Dependencies: nil,
			Mode:         workflow.RunModeSerial,
			Policy:       workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort},
		},
	}

	// 执行
	ctx := context.Background()
	result, err := orchestrator.Execute(ctx, definitions)

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Len(t, result.TaskResults, 1)
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	orchestrator := NewTaskOrchestrator()

	// 创建会失败的任务
	failingTask := task.NewFuncTask("failing-task", "Failing Task",
		func(ctx context.Context) (interface{}, error) {
			return nil, fmt.Errorf("task failed")
		})

	definitions := []TaskDefinition{
		{
			ID:           "failing-task",
			Name:         "Failing Task",
			Task:         failingTask,
			Dependencies: nil,
			Mode:         workflow.RunModeSerial,
			Policy:       workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort},
		},
	}

	// 执行
	ctx := context.Background()
	result, err := orchestrator.Execute(ctx, definitions)

	assert.Error(t, err)
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Error)
}

// TestSkipFailurePolicy 测试跳过失败策略
func TestSkipFailurePolicy(t *testing.T) {
	orchestrator := NewTaskOrchestrator()

	// 创建会失败的任务和正常任务
	failingTask := task.NewFuncTask("failing-task", "Failing Task",
		func(ctx context.Context) (interface{}, error) {
			return nil, fmt.Errorf("task failed")
		})

	successTask := task.NewFuncTask("success-task", "Success Task",
		func(ctx context.Context) (interface{}, error) {
			return "success", nil
		})

	definitions := []TaskDefinition{
		{
			ID:           "failing-task",
			Name:         "Failing Task",
			Task:         failingTask,
			Dependencies: nil,
			Mode:         workflow.RunModeSerial,
			Policy:       workflow.ExecutionPolicy{OnFailure: workflow.FailureSkip}, // 跳过失败
		},
		{
			ID:           "success-task",
			Name:         "Success Task",
			Task:         successTask,
			Dependencies: []string{"failing-task"},
			Mode:         workflow.RunModeSerial,
			Policy:       workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort},
		},
	}

	// 执行
	ctx := context.Background()
	result, err := orchestrator.Execute(ctx, definitions)

	// 由于使用了Skip策略，整体应该成功
	assert.NoError(t, err)
	assert.True(t, result.Success)
}

// TestValidationErrors 测试验证错误
func TestValidationErrors(t *testing.T) {
	orchestrator := NewTaskOrchestrator()

	// 测试空定义
	_, err := orchestrator.Execute(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "任务定义不能为空")

	// 测试重复ID
	task1 := task.NewFuncTask("duplicate", "Task 1", func(ctx context.Context) (interface{}, error) {
		return nil, nil
	})
	task2 := task.NewFuncTask("duplicate", "Task 2", func(ctx context.Context) (interface{}, error) {
		return nil, nil
	})

	definitions := []TaskDefinition{
		{ID: "duplicate", Task: task1},
		{ID: "duplicate", Task: task2}, // 重复ID
	}

	_, err = orchestrator.Execute(context.Background(), definitions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "任务ID重复")

	// 测试无效依赖
	definitions = []TaskDefinition{
		{
			ID:           "task1",
			Task:         task1,
			Dependencies: []string{"nonexistent"}, // 不存在的依赖
		},
	}

	_, err = orchestrator.Execute(context.Background(), definitions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "依赖的任务")
}

// TestWithHooks 测试Hook集成
func TestWithHooks(t *testing.T) {
	// 创建日志Hook
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	loggingHook := executorx.NewLoggingHook(logrus.NewEntry(logger))

	// 创建指标Hook
	metricsHook := executorx.NewMetricsHook()

	orchestrator := NewTaskOrchestrator().
		AddHook(loggingHook).
		AddHook(metricsHook)

	// 创建简单任务
	testTask := task.NewFuncTask("hook-task", "Hook Task",
		func(ctx context.Context) (interface{}, error) {
			return "success", nil
		})

	definitions := []TaskDefinition{
		{
			ID:   "hook-task",
			Task: testTask,
		},
	}

	// 执行
	ctx := context.Background()
	result, err := orchestrator.Execute(ctx, definitions)

	assert.NoError(t, err)
	assert.True(t, result.Success)

	// Hook应该已经被调用（通过日志可以验证）
}

// TestCreateSimpleWorkflow 测试简单工作流创建
func TestCreateSimpleWorkflow(t *testing.T) {
	orchestrator := NewTaskOrchestrator()

	tasks := []task.Task{
		task.NewFuncTask("task1", "Task 1", func(ctx context.Context) (interface{}, error) { return "1", nil }),
		task.NewFuncTask("task2", "Task 2", func(ctx context.Context) (interface{}, error) { return "2", nil }),
		task.NewFuncTask("task3", "Task 3", func(ctx context.Context) (interface{}, error) { return "3", nil }),
	}

	definitions, err := orchestrator.CreateSimpleWorkflow(tasks...)
	assert.NoError(t, err)
	assert.Len(t, definitions, 3)

	// 验证依赖关系
	assert.Empty(t, definitions[0].Dependencies)                    // 第一个任务无依赖
	assert.Equal(t, []string{"task1"}, definitions[1].Dependencies) // 第二个任务依赖第一个
	assert.Equal(t, []string{"task2"}, definitions[2].Dependencies) // 第三个任务依赖第二个
}

// TestComplexWorkflow 测试复杂工作流
func TestComplexWorkflow(t *testing.T) {
	orchestrator := NewTaskOrchestrator()

	// 创建复杂的依赖关系: A -> [B, C] -> D
	taskA := task.NewFuncTask("A", "Task A", func(ctx context.Context) (interface{}, error) {
		time.Sleep(50 * time.Millisecond)
		return "A完成", nil
	})

	taskB := task.NewFuncTask("B", "Task B", func(ctx context.Context) (interface{}, error) {
		time.Sleep(30 * time.Millisecond)
		return "B完成", nil
	})

	taskC := task.NewFuncTask("C", "Task C", func(ctx context.Context) (interface{}, error) {
		time.Sleep(40 * time.Millisecond)
		return "C完成", nil
	})

	taskD := task.NewFuncTask("D", "Task D", func(ctx context.Context) (interface{}, error) {
		time.Sleep(20 * time.Millisecond)
		return "D完成", nil
	})

	definitions := []TaskDefinition{
		{ID: "A", Task: taskA, Mode: workflow.RunModeSerial, Policy: workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort}},
		{ID: "B", Task: taskB, Dependencies: []string{"A"}, Mode: workflow.RunModeParallel, Policy: workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort}},
		{ID: "C", Task: taskC, Dependencies: []string{"A"}, Mode: workflow.RunModeParallel, Policy: workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort}},
		{ID: "D", Task: taskD, Dependencies: []string{"B", "C"}, Mode: workflow.RunModeSerial, Policy: workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort}},
	}

	ctx := context.Background()
	start := time.Now()
	result, err := orchestrator.Execute(ctx, definitions)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Len(t, result.TaskResults, 4)

	// 验证执行时间（B和C应该并行执行）
	// 预期时间: A(50ms) + max(B(30ms), C(40ms)) + D(20ms) = ~110ms
	assert.True(t, duration < 200*time.Millisecond, "复杂工作流应该在合理时间内完成")

	t.Logf("复杂工作流执行时间: %v", duration)
}

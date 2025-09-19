package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fengmingli/orchestrator/internal/engine/executorx"
	"github.com/fengmingli/orchestrator/internal/engine/task"
	"github.com/fengmingli/orchestrator/internal/engine/workflow"
	"github.com/sirupsen/logrus"
)

// TaskOrchestrator 任务编排器，集成workflow和executor
type TaskOrchestrator struct {
	executor   *executorx.RetryableExecutor
	logger     *logrus.Entry
	maxWorkers int
	hooks      []executorx.Hook
	mu         sync.RWMutex
}

// NewTaskOrchestrator 创建新的任务编排器
func NewTaskOrchestrator() *TaskOrchestrator {
	logger := logrus.NewEntry(logrus.New())
	executor := executorx.NewRetryableExecutor().
		WithLogger(logger).
		WithMaxRetries(3).
		WithRetryDelay(time.Second)

	return &TaskOrchestrator{
		executor:   executor,
		logger:     logger,
		maxWorkers: 10,
		hooks:      make([]executorx.Hook, 0),
	}
}

// WithLogger 设置日志记录器
func (o *TaskOrchestrator) WithLogger(logger *logrus.Entry) *TaskOrchestrator {
	if logger != nil {
		o.logger = logger
		o.executor.WithLogger(logger)
	}
	return o
}

// WithMaxWorkers 设置最大工作者数量
func (o *TaskOrchestrator) WithMaxWorkers(workers int) *TaskOrchestrator {
	if workers > 0 {
		o.maxWorkers = workers
	}
	return o
}

// WithRetryConfig 设置重试配置
func (o *TaskOrchestrator) WithRetryConfig(maxRetries uint, retryDelay time.Duration) *TaskOrchestrator {
	o.executor.WithMaxRetries(maxRetries).WithRetryDelay(retryDelay)
	return o
}

// AddHook 添加Hook
func (o *TaskOrchestrator) AddHook(hook executorx.Hook) *TaskOrchestrator {
	o.mu.Lock()
	defer o.mu.Unlock()

	if hook != nil {
		o.hooks = append(o.hooks, hook)

		// 如果只有一个hook，直接设置
		if len(o.hooks) == 1 {
			o.executor.WithHook(hook)
		} else {
			// 如果有多个hook，使用组合hook
			compositeHook := executorx.NewCompositeHook(o.hooks...)
			o.executor.WithHook(compositeHook)
		}
	}
	return o
}

// TaskDefinition 任务定义
type TaskDefinition struct {
	ID           string                   `json:"id"`
	Name         string                   `json:"name"`
	Task         task.Task                `json:"-"` // 不序列化
	Dependencies []string                 `json:"dependencies"`
	Mode         workflow.RunMode         `json:"mode"`
	Policy       workflow.ExecutionPolicy `json:"policy"`
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Success     bool                       `json:"success"`
	Error       string                     `json:"error,omitempty"`
	TaskResults map[string]task.ExecResult `json:"task_results"`
	Duration    time.Duration              `json:"duration"`
	StartTime   time.Time                  `json:"start_time"`
	FinishTime  time.Time                  `json:"finish_time"`
}

// Execute 执行任务编排
func (o *TaskOrchestrator) Execute(ctx context.Context, definitions []TaskDefinition) (*ExecutionResult, error) {
	start := time.Now()
	result := &ExecutionResult{
		TaskResults: make(map[string]task.ExecResult),
		StartTime:   start,
	}

	o.logger.WithField("task_count", len(definitions)).Info("开始执行任务编排")

	// 验证任务定义
	if err := o.validateDefinitions(definitions); err != nil {
		result.Success = false
		result.Error = err.Error()
		result.FinishTime = time.Now()
		result.Duration = result.FinishTime.Sub(start)
		return result, err
	}

	// 构建workflow描述
	descs := make([]workflow.Desc, 0, len(definitions))
	taskMap := make(map[string]task.Task)

	for _, def := range definitions {
		taskMap[def.ID] = def.Task

		// 将task包装成workflow可执行的runner
		runner := o.createTaskRunner(ctx, def.Task, def.ID)

		descs = append(descs, workflow.Desc{
			ID:     def.ID,
			Mode:   def.Mode,
			Deps:   def.Dependencies,
			Runner: runner,
			Policy: def.Policy,
		})
	}

	// 创建DAG
	dag, err := workflow.NewDAG(descs)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("创建DAG失败: %v", err)
		result.FinishTime = time.Now()
		result.Duration = result.FinishTime.Sub(start)
		return result, err
	}

	// 创建调度器并执行
	scheduler := workflow.NewScheduler(dag, o.maxWorkers)

	o.logger.Info("开始执行DAG调度")
	execErr := scheduler.Run(ctx, nil)

	// 收集任务结果
	o.collectTaskResults(result, taskMap)

	result.FinishTime = time.Now()
	result.Duration = result.FinishTime.Sub(start)

	if execErr != nil {
		result.Success = false
		result.Error = execErr.Error()
		o.logger.WithError(execErr).Error("DAG执行失败")
		return result, execErr
	}

	result.Success = true
	o.logger.WithField("duration", result.Duration).Info("任务编排执行完成")

	return result, nil
}

// createTaskRunner 创建任务运行器
func (o *TaskOrchestrator) createTaskRunner(ctx context.Context, taskInst task.Task, taskID string) func() error {
	return executorx.WrapTask(ctx, taskInst, o.executor, taskID)
}

// validateDefinitions 验证任务定义
func (o *TaskOrchestrator) validateDefinitions(definitions []TaskDefinition) error {
	if len(definitions) == 0 {
		return fmt.Errorf("任务定义不能为空")
	}

	// 检查任务ID唯一性
	idSet := make(map[string]bool)
	for _, def := range definitions {
		if def.ID == "" {
			return fmt.Errorf("任务ID不能为空")
		}
		if idSet[def.ID] {
			return fmt.Errorf("任务ID重复: %s", def.ID)
		}
		idSet[def.ID] = true

		if def.Task == nil {
			return fmt.Errorf("任务%s的Task不能为空", def.ID)
		}

		// 验证任务本身
		if err := def.Task.Validate(); err != nil {
			return fmt.Errorf("任务%s验证失败: %v", def.ID, err)
		}
	}

	// 检查依赖关系的有效性
	for _, def := range definitions {
		for _, depID := range def.Dependencies {
			if !idSet[depID] {
				return fmt.Errorf("任务%s依赖的任务%s不存在", def.ID, depID)
			}
		}
	}

	return nil
}

// collectTaskResults 收集任务结果
func (o *TaskOrchestrator) collectTaskResults(result *ExecutionResult, taskMap map[string]task.Task) {
	// 这里可以从executor的结果中收集具体的任务执行结果
	// 目前简化处理，实际使用时可以增强这个功能
	for id := range taskMap {
		result.TaskResults[id] = task.ExecResult{
			TaskID: id,
			// 其他字段可以从实际执行结果中获取
		}
	}
}

// CreateSimpleWorkflow 创建简单的工作流
func (o *TaskOrchestrator) CreateSimpleWorkflow(tasks ...task.Task) ([]TaskDefinition, error) {
	definitions := make([]TaskDefinition, 0, len(tasks))

	for i, t := range tasks {
		def := TaskDefinition{
			ID:           t.GetID(),
			Name:         t.GetName(),
			Task:         t,
			Dependencies: nil,                      // 简单工作流无依赖
			Mode:         workflow.RunModeParallel, // 默认并行
			Policy: workflow.ExecutionPolicy{
				OnFailure: workflow.FailureAbort,
			},
		}

		// 如果有前一个任务，添加依赖关系（串行执行）
		if i > 0 {
			def.Dependencies = []string{tasks[i-1].GetID()}
			def.Mode = workflow.RunModeSerial
		}

		definitions = append(definitions, def)
	}

	return definitions, nil
}

// CreateParallelWorkflow 创建并行工作流
func (o *TaskOrchestrator) CreateParallelWorkflow(tasks ...task.Task) ([]TaskDefinition, error) {
	definitions := make([]TaskDefinition, 0, len(tasks))

	for _, t := range tasks {
		def := TaskDefinition{
			ID:           t.GetID(),
			Name:         t.GetName(),
			Task:         t,
			Dependencies: nil, // 并行工作流无依赖
			Mode:         workflow.RunModeParallel,
			Policy: workflow.ExecutionPolicy{
				OnFailure: workflow.FailureAbort,
			},
		}
		definitions = append(definitions, def)
	}

	return definitions, nil
}

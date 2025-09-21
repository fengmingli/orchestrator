package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fengmingli/orchestrator/internal/engine"
	"github.com/fengmingli/orchestrator/internal/engine/task"
	"github.com/fengmingli/orchestrator/internal/engine/workflow"
	"github.com/fengmingli/orchestrator/internal/lock"
	"github.com/fengmingli/orchestrator/internal/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// OrchestratorService 编排服务，集成workflow引擎
type OrchestratorService struct {
	db               *gorm.DB
	executionService *ExecutionService
	orchestrator     *engine.TaskOrchestrator
	lockManager      *lock.LockManager
	logger           *logrus.Entry
}

// NewOrchestratorService 创建编排服务
func NewOrchestratorService(db *gorm.DB) *OrchestratorService {
	return NewOrchestratorServiceWithLockConfig(db, nil)
}

// NewOrchestratorServiceWithLockConfig 使用指定锁配置创建编排服务
func NewOrchestratorServiceWithLockConfig(db *gorm.DB, lockConfig *lock.LockConfig) *OrchestratorService {
	logger := logrus.NewEntry(logrus.New())
	
	// 创建任务编排器
	orchestrator := engine.NewTaskOrchestrator().
		WithLogger(logger).
		WithMaxWorkers(10).
		WithRetryConfig(3, time.Second)

	// 创建分布式锁管理器
	lockManager, err := lock.NewLockManager(db, lockConfig)
	if err != nil {
		logger.WithError(err).Fatal("创建锁管理器失败")
	}

	return &OrchestratorService{
		db:               db,
		executionService: NewExecutionService(db),
		orchestrator:     orchestrator,
		lockManager:      lockManager,
		logger:           logger,
	}
}

// ExecuteWorkflow 执行工作流
func (s *OrchestratorService) ExecuteWorkflow(executionID string) error {
	ctx := context.Background()
	logger := s.logger.WithField("execution_id", executionID)
	
	logger.Info("开始执行工作流")

	// 获取执行记录
	execution, err := s.executionService.GetExecution(executionID)
	if err != nil {
		return fmt.Errorf("获取执行记录失败: %w", err)
	}

	// 检查执行状态
	if execution.Status != "pending" {
		return fmt.Errorf("执行状态不是pending，无法启动")
	}

	// 获取分布式锁，确保只有一个副本执行
	workflowProvider := s.lockManager.GetWorkflowLockProvider()
	workflowLock, err := workflowProvider.LockWorkflowExecution(ctx, executionID)
	if err != nil {
		if errors.Is(err, lock.ErrWorkflowAlreadyRunning) {
			logger.Info("工作流已被其他副本执行，跳过执行")
			return nil // 不是错误，只是跳过执行
		}
		logger.WithError(err).Error("获取工作流执行锁失败")
		return fmt.Errorf("获取工作流执行锁失败: %w", err)
	}

	// 确保在函数结束时释放锁
	defer func() {
		if unlockErr := workflowLock.Unlock(ctx); unlockErr != nil {
			logger.WithError(unlockErr).Error("释放工作流锁失败")
		}
	}()

	// 更新执行状态为运行中
	startTime := time.Now()
	if err := s.executionService.UpdateExecutionStatus(executionID, &model.ExecutionStatusUpdate{
		Status:    "running",
		StartedAt: &startTime,
	}); err != nil {
		return fmt.Errorf("更新执行状态失败: %w", err)
	}

	// 构建任务定义
	taskDefinitions, err := s.buildTaskDefinitions(execution)
	if err != nil {
		s.markExecutionFailed(executionID, err)
		return fmt.Errorf("构建任务定义失败: %w", err)
	}

	// 执行工作流前刷新锁的过期时间
	if err := workflowLock.Refresh(ctx, 10*time.Minute); err != nil {
		logger.WithError(err).Warn("刷新工作流锁失败")
	}

	// 执行工作流
	result, err := s.orchestrator.Execute(ctx, taskDefinitions)

	// 更新执行结果
	finishTime := time.Now()
	if err != nil {
		s.logger.WithError(err).Error("工作流执行失败")
		s.executionService.UpdateExecutionStatus(executionID, &model.ExecutionStatusUpdate{
			Status:     "failed",
			Error:      err.Error(),
			FinishedAt: &finishTime,
		})
		return err
	}

	// 更新成功状态
	status := "success"
	if !result.Success {
		status = "failed"
	}

	s.executionService.UpdateExecutionStatus(executionID, &model.ExecutionStatusUpdate{
		Status:     status,
		Error:      result.Error,
		FinishedAt: &finishTime,
	})

	s.logger.WithFields(logrus.Fields{
		"execution_id": executionID,
		"status":       status,
		"duration":     result.Duration,
	}).Info("工作流执行完成")

	return nil
}

// buildTaskDefinitions 构建任务定义
func (s *OrchestratorService) buildTaskDefinitions(execution *model.WorkflowExecution) ([]engine.TaskDefinition, error) {
	// 获取模板步骤
	var templateSteps []model.WorkflowTemplateStep
	if err := s.db.Preload("Step").Where("template_id = ?", execution.TemplateID).Order("\"order\" asc").Find(&templateSteps).Error; err != nil {
		return nil, fmt.Errorf("获取模板步骤失败: %w", err)
	}

	definitions := make([]engine.TaskDefinition, 0, len(templateSteps))

	for _, templateStep := range templateSteps {
		step := templateStep.Step
		if step == nil {
			return nil, fmt.Errorf("步骤 %s 不存在", templateStep.StepID)
		}

		// 创建具体的任务实例
		taskInstance, err := s.createTaskInstance(step, execution.ID)
		if err != nil {
			return nil, fmt.Errorf("创建任务实例失败: %w", err)
		}

		// 解析依赖关系
		var dependencies []string
		if templateStep.Dependencies != "" {
			if err := json.Unmarshal([]byte(templateStep.Dependencies), &dependencies); err != nil {
				return nil, fmt.Errorf("解析依赖关系失败: %w", err)
			}
		}

		// 转换运行模式
		runMode := workflow.RunModeSerial
		if templateStep.RunMode == "parallel" {
			runMode = workflow.RunModeParallel
		}

		// 转换失败策略
		failurePolicy := workflow.FailureAbort
		if templateStep.OnFailure == "skip" {
			failurePolicy = workflow.FailureSkip
		}

		definition := engine.TaskDefinition{
			ID:           step.ID,
			Name:         step.Name,
			Task:         taskInstance,
			Dependencies: dependencies,
			Mode:         runMode,
			Policy: workflow.ExecutionPolicy{
				OnFailure: failurePolicy,
			},
		}

		definitions = append(definitions, definition)
	}

	return definitions, nil
}

// createTaskInstance 创建任务实例
func (s *OrchestratorService) createTaskInstance(step *model.Step, executionID string) (task.Task, error) {
	switch step.ExecutorType {
	case "http":
		httpTask := task.NewHTTPTask(step.ID, step.Name, step.HTTPMethod, step.HTTPURL)
		
		// 设置请求头
		if step.HTTPHeaders != "" {
			var headers map[string]string
			if err := json.Unmarshal([]byte(step.HTTPHeaders), &headers); err != nil {
				return nil, fmt.Errorf("解析HTTP请求头失败: %w", err)
			}
			httpTask.SetHeaders(headers)
		}
		
		// 设置请求体
		if step.HTTPBody != "" {
			httpTask.SetBody(step.HTTPBody)
		}
		
		// 设置超时
		if step.Timeout > 0 {
			httpTask.SetTimeout(time.Duration(step.Timeout) * time.Second)
		}
		
		// 包装任务以支持状态回调
		return s.wrapTaskWithCallback(httpTask, step.ID, executionID), nil

	case "shell":
		shellTask := task.NewShellTask(step.ID, step.Name, step.ShellScript)
		
		// 设置环境变量
		if step.ShellEnv != "" {
			var env map[string]string
			if err := json.Unmarshal([]byte(step.ShellEnv), &env); err != nil {
				return nil, fmt.Errorf("解析环境变量失败: %w", err)
			}
			shellTask.SetEnv(env)
		}
		
		// 设置超时
		if step.Timeout > 0 {
			shellTask.SetTimeout(time.Duration(step.Timeout) * time.Second)
		}
		
		return s.wrapTaskWithCallback(shellTask, step.ID, executionID), nil

	case "func":
		// 函数任务，执行一个简单的回调函数
		funcTask := task.NewFuncTask(step.ID, step.Name, func(ctx context.Context) (interface{}, error) {
			s.logger.WithFields(logrus.Fields{
				"step_id":      step.ID,
				"execution_id": executionID,
			}).Info("执行函数任务")
			
			return fmt.Sprintf("函数任务 %s 执行完成", step.Name), nil
		})
		
		return s.wrapTaskWithCallback(funcTask, step.ID, executionID), nil

	default:
		return nil, fmt.Errorf("不支持的执行器类型: %s", step.ExecutorType)
	}
}

// wrapTaskWithCallback 包装任务以支持状态回调
func (s *OrchestratorService) wrapTaskWithCallback(baseTask task.Task, stepID, executionID string) task.Task {
	return &callbackTask{
		Task:             baseTask,
		stepID:           stepID,
		executionID:      executionID,
		executionService: s.executionService,
		logger:           s.logger,
	}
}

// markExecutionFailed 标记执行失败
func (s *OrchestratorService) markExecutionFailed(executionID string, err error) {
	finishTime := time.Now()
	s.executionService.UpdateExecutionStatus(executionID, &model.ExecutionStatusUpdate{
		Status:     "failed",
		Error:      err.Error(),
		FinishedAt: &finishTime,
	})
}

// callbackTask 回调任务包装器
type callbackTask struct {
	task.Task
	stepID           string
	executionID      string
	executionService *ExecutionService
	logger           *logrus.Entry
}

// Execute 执行任务并记录状态
func (ct *callbackTask) Execute(ctx context.Context) (task.ExecResult, error) {
	// 更新开始状态
	startTime := time.Now()
	ct.executionService.UpdateStepExecutionStatus(ct.executionID, ct.stepID, &model.StepExecutionStatusUpdate{
		Status:    "running",
		StartedAt: &startTime,
	})

	ct.logger.WithFields(logrus.Fields{
		"step_id":      ct.stepID,
		"execution_id": ct.executionID,
	}).Info("步骤开始执行")

	// 执行原始任务
	result, err := ct.Task.Execute(ctx)
	finishTime := time.Now()

	// 更新执行结果
	if err != nil {
		ct.logger.WithError(err).WithFields(logrus.Fields{
			"step_id":      ct.stepID,
			"execution_id": ct.executionID,
		}).Error("步骤执行失败")

		ct.executionService.UpdateStepExecutionStatus(ct.executionID, ct.stepID, &model.StepExecutionStatusUpdate{
			Status:     "failed",
			Error:      err.Error(),
			FinishedAt: &finishTime,
		})
	} else {
		ct.logger.WithFields(logrus.Fields{
			"step_id":      ct.stepID,
			"execution_id": ct.executionID,
			"duration":     result.Duration,
		}).Info("步骤执行成功")

		output := ""
		if result.Output != nil {
			if str, ok := result.Output.(string); ok {
				output = str
			} else {
				outputBytes, _ := json.Marshal(result.Output)
				output = string(outputBytes)
			}
		}

		ct.executionService.UpdateStepExecutionStatus(ct.executionID, ct.stepID, &model.StepExecutionStatusUpdate{
			Status:     "success",
			Output:     output,
			FinishedAt: &finishTime,
		})
	}

	return result, err
}
package service

import (
	"fmt"
	"time"

	"github.com/fengmingli/orchestrator/internal/model"
	"gorm.io/gorm"
)

// ExecutionService 执行服务
type ExecutionService struct {
	db *gorm.DB
}

// NewExecutionService 创建执行服务
func NewExecutionService(db *gorm.DB) *ExecutionService {
	return &ExecutionService{db: db}
}

// CreateExecution 创建执行
func (s *ExecutionService) CreateExecution(req *model.ExecutionCreateRequest) (*model.WorkflowExecution, error) {
	// 检查模板是否存在
	var template model.WorkflowTemplate
	if err := s.db.Where("id = ? AND is_active = ?", req.TemplateID, true).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("模板不存在或已禁用")
		}
		return nil, fmt.Errorf("获取模板失败: %w", err)
	}

	execution := &model.WorkflowExecution{
		TemplateID: req.TemplateID,
		Status:     "pending",
		CreatedBy:  req.CreatedBy,
	}

	if err := s.db.Create(execution).Error; err != nil {
		return nil, fmt.Errorf("创建执行失败: %w", err)
	}

	// 初始化步骤执行记录
	if err := s.initStepExecutions(execution.ID, req.TemplateID); err != nil {
		return nil, fmt.Errorf("初始化步骤执行记录失败: %w", err)
	}

	return s.GetExecution(execution.ID)
}

// ListExecutions 获取执行列表  
func (s *ExecutionService) ListExecutions(page, size int, templateID, status string) ([]*model.ExecutionSummary, int64, error) {
	var executions []*model.ExecutionSummary
	var total int64

	query := s.db.Table("workflow_executions e").
		Select(`e.id, e.template_id, t.name as template_name, e.status, 
				e.started_at, e.finished_at, e.duration, e.created_by, e.created_at,
				COUNT(se.id) as total_steps,
				SUM(CASE WHEN se.status = 'success' THEN 1 ELSE 0 END) as success_steps,
				SUM(CASE WHEN se.status = 'failed' THEN 1 ELSE 0 END) as failed_steps`).
		Joins("LEFT JOIN workflow_templates t ON e.template_id = t.id").
		Joins("LEFT JOIN workflow_step_executions se ON e.id = se.execution_id").
		Group("e.id")

	// 筛选条件
	if templateID != "" {
		query = query.Where("e.template_id = ?", templateID)
	}
	if status != "" {
		query = query.Where("e.status = ?", status)
	}

	// 计算总数
	countQuery := s.db.Model(&model.WorkflowExecution{})
	if templateID != "" {
		countQuery = countQuery.Where("template_id = ?", templateID)
	}
	if status != "" {
		countQuery = countQuery.Where("status = ?", status)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取执行总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("e.created_at desc").Find(&executions).Error; err != nil {
		return nil, 0, fmt.Errorf("获取执行列表失败: %w", err)
	}

	return executions, total, nil
}

// GetExecution 获取执行详情
func (s *ExecutionService) GetExecution(id string) (*model.WorkflowExecution, error) {
	var execution model.WorkflowExecution
	if err := s.db.Preload("Template").Preload("Steps.Step").Where("id = ?", id).First(&execution).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("执行不存在")
		}
		return nil, fmt.Errorf("获取执行失败: %w", err)
	}

	return &execution, nil
}

// UpdateExecutionStatus 更新执行状态
func (s *ExecutionService) UpdateExecutionStatus(id string, update *model.ExecutionStatusUpdate) error {
	updates := map[string]interface{}{
		"status": update.Status,
	}

	if update.Error != "" {
		updates["error"] = update.Error
	}
	if update.StartedAt != nil {
		updates["started_at"] = update.StartedAt
	}
	if update.FinishedAt != nil {
		updates["finished_at"] = update.FinishedAt
		// 计算执行时长
		execution, err := s.GetExecution(id)
		if err == nil && execution.StartedAt != nil {
			duration := update.FinishedAt.Sub(*execution.StartedAt)
			updates["duration"] = duration.Milliseconds()
		}
	}

	if err := s.db.Model(&model.WorkflowExecution{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新执行状态失败: %w", err)
	}

	return nil
}

// UpdateStepExecutionStatus 更新步骤执行状态
func (s *ExecutionService) UpdateStepExecutionStatus(executionID, stepID string, update *model.StepExecutionStatusUpdate) error {
	updates := map[string]interface{}{
		"status": update.Status,
	}

	if update.Output != "" {
		updates["output"] = update.Output
	}
	if update.Error != "" {
		updates["error"] = update.Error
	}
	if update.StartedAt != nil {
		updates["started_at"] = update.StartedAt
	}
	if update.FinishedAt != nil {
		updates["finished_at"] = update.FinishedAt
		// 计算执行时长
		var stepExecution model.WorkflowStepExecution
		if err := s.db.Where("execution_id = ? AND step_id = ?", executionID, stepID).First(&stepExecution).Error; err == nil && stepExecution.StartedAt != nil {
			duration := update.FinishedAt.Sub(*stepExecution.StartedAt)
			updates["duration"] = duration.Milliseconds()
		}
	}
	if update.RetryCount > 0 {
		updates["retry_count"] = update.RetryCount
	}

	if err := s.db.Model(&model.WorkflowStepExecution{}).Where("execution_id = ? AND step_id = ?", executionID, stepID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新步骤执行状态失败: %w", err)
	}

	return nil
}

// CancelExecution 取消执行
func (s *ExecutionService) CancelExecution(id string) error {
	now := time.Now()
	return s.UpdateExecutionStatus(id, &model.ExecutionStatusUpdate{
		Status:     "cancelled",
		FinishedAt: &now,
	})
}

// GetExecutionLogs 获取执行日志
func (s *ExecutionService) GetExecutionLogs(id string) ([]*model.WorkflowStepExecution, error) {
	var stepExecutions []*model.WorkflowStepExecution
	if err := s.db.Preload("Step").Where("execution_id = ?", id).Order("created_at asc").Find(&stepExecutions).Error; err != nil {
		return nil, fmt.Errorf("获取执行日志失败: %w", err)
	}

	return stepExecutions, nil
}

// GetExecutionStatus 获取执行状态
func (s *ExecutionService) GetExecutionStatus(id string) (map[string]interface{}, error) {
	execution, err := s.GetExecution(id)
	if err != nil {
		return nil, err
	}

	// 统计步骤状态
	var stepStats []struct {
		Status string
		Count  int
	}
	if err := s.db.Model(&model.WorkflowStepExecution{}).
		Select("status, count(*) as count").
		Where("execution_id = ?", id).
		Group("status").
		Find(&stepStats).Error; err != nil {
		return nil, fmt.Errorf("获取步骤统计失败: %w", err)
	}

	statsMap := make(map[string]int)
	for _, stat := range stepStats {
		statsMap[stat.Status] = stat.Count
	}

	return map[string]interface{}{
		"execution":   execution,
		"step_stats":  statsMap,
		"total_steps": len(execution.Steps),
		"progress":    s.calculateProgress(statsMap, len(execution.Steps)),
	}, nil
}

// initStepExecutions 初始化步骤执行记录
func (s *ExecutionService) initStepExecutions(executionID, templateID string) error {
	// 获取模板步骤
	var templateSteps []model.WorkflowTemplateStep
	if err := s.db.Where("template_id = ?", templateID).Find(&templateSteps).Error; err != nil {
		return fmt.Errorf("获取模板步骤失败: %w", err)
	}

	// 创建步骤执行记录
	for _, templateStep := range templateSteps {
		stepExecution := &model.WorkflowStepExecution{
			ExecutionID: executionID,
			StepID:      templateStep.StepID,
			Status:      "pending",
		}

		if err := s.db.Create(stepExecution).Error; err != nil {
			return fmt.Errorf("创建步骤执行记录失败: %w", err)
		}
	}

	return nil
}

// calculateProgress 计算执行进度
func (s *ExecutionService) calculateProgress(stats map[string]int, total int) float64 {
	if total == 0 {
		return 0
	}

	completed := stats["success"] + stats["failed"] + stats["skipped"]
	return float64(completed) / float64(total) * 100
}
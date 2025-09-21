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
	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查模板是否存在
	var template model.WorkflowTemplate
	if err := tx.Where("id = ? AND is_active = ?", req.TemplateID, true).First(&template).Error; err != nil {
		tx.Rollback()
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

	if err := tx.Create(execution).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建执行失败: %w", err)
	}

	// 初始化步骤执行记录
	if err := s.initStepExecutionsWithTx(tx, execution.ID, req.TemplateID); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("初始化步骤执行记录失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
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
	if err := s.db.Where("id = ?", id).First(&execution).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("执行不存在")
		}
		return nil, fmt.Errorf("获取执行失败: %w", err)
	}

	// 手动加载关联数据
	if err := s.loadExecutionDetails(&execution); err != nil {
		return nil, fmt.Errorf("加载执行详情失败: %w", err)
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
	if err := s.db.Where("execution_id = ?", id).Order("created_at asc").Find(&stepExecutions).Error; err != nil {
		return nil, fmt.Errorf("获取执行日志失败: %w", err)
	}

	// 手动加载步骤信息
	if err := s.loadStepExecutionDetails(stepExecutions); err != nil {
		return nil, fmt.Errorf("加载步骤详情失败: %w", err)
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

// initStepExecutionsWithTx 使用事务初始化步骤执行记录
func (s *ExecutionService) initStepExecutionsWithTx(tx *gorm.DB, executionID, templateID string) error {
	// 获取模板步骤
	var templateSteps []model.WorkflowTemplateStep
	if err := tx.Where("template_id = ?", templateID).Find(&templateSteps).Error; err != nil {
		return fmt.Errorf("获取模板步骤失败: %w", err)
	}

	// 验证模板步骤是否存在
	if len(templateSteps) == 0 {
		return fmt.Errorf("模板没有定义步骤")
	}

	// 收集所有步骤ID，验证步骤是否存在
	stepIDs := make([]string, 0, len(templateSteps))
	for _, ts := range templateSteps {
		stepIDs = append(stepIDs, ts.StepID)
	}

	var existingStepsCount int64
	if err := tx.Model(&model.Step{}).Where("id IN ?", stepIDs).Count(&existingStepsCount).Error; err != nil {
		return fmt.Errorf("验证步骤存在性失败: %w", err)
	}

	if int(existingStepsCount) != len(stepIDs) {
		return fmt.Errorf("部分步骤不存在，无法创建执行")
	}

	// 创建步骤执行记录
	for _, templateStep := range templateSteps {
		stepExecution := &model.WorkflowStepExecution{
			ExecutionID: executionID,
			StepID:      templateStep.StepID,
			Status:      "pending",
		}

		if err := tx.Create(stepExecution).Error; err != nil {
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

// loadExecutionDetails 手动加载执行详情
func (s *ExecutionService) loadExecutionDetails(execution *model.WorkflowExecution) error {
	// 加载模板信息
	var template model.WorkflowTemplate
	if err := s.db.Where("id = ?", execution.TemplateID).First(&template).Error; err != nil {
		return fmt.Errorf("加载模板信息失败: %w", err)
	}
	execution.Template = &template

	// 加载步骤执行记录
	var stepExecutions []model.WorkflowStepExecution
	if err := s.db.Where("execution_id = ?", execution.ID).Order("created_at asc").Find(&stepExecutions).Error; err != nil {
		return fmt.Errorf("加载步骤执行记录失败: %w", err)
	}

	// 加载步骤执行记录的详情
	stepExecutionPtrs := make([]*model.WorkflowStepExecution, len(stepExecutions))
	for i := range stepExecutions {
		stepExecutionPtrs[i] = &stepExecutions[i]
	}
	if err := s.loadStepExecutionDetails(stepExecutionPtrs); err != nil {
		return fmt.Errorf("加载步骤执行详情失败: %w", err)
	}

	execution.Steps = stepExecutions
	return nil
}

// loadStepExecutionDetails 手动加载步骤执行详情
func (s *ExecutionService) loadStepExecutionDetails(stepExecutions []*model.WorkflowStepExecution) error {
	if len(stepExecutions) == 0 {
		return nil
	}

	// 收集所有步骤ID
	stepIDs := make([]string, 0, len(stepExecutions))
	for _, se := range stepExecutions {
		stepIDs = append(stepIDs, se.StepID)
	}

	// 批量查询步骤信息
	var steps []model.Step
	if err := s.db.Where("id IN ?", stepIDs).Find(&steps).Error; err != nil {
		return fmt.Errorf("查询步骤信息失败: %w", err)
	}

	// 创建步骤映射
	stepMap := make(map[string]*model.Step)
	for i := range steps {
		stepMap[steps[i].ID] = &steps[i]
	}

	// 填充Step字段
	for _, se := range stepExecutions {
		if step, exists := stepMap[se.StepID]; exists {
			se.Step = step
		}
	}

	return nil
}

// DeleteExecution 删除执行记录（包括所有相关的步骤执行记录）
func (s *ExecutionService) DeleteExecution(id string) error {
	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查执行是否存在
	var execution model.WorkflowExecution
	if err := tx.Where("id = ?", id).First(&execution).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("执行记录不存在")
		}
		return fmt.Errorf("获取执行记录失败: %w", err)
	}

	// 检查执行状态，如果正在运行则不允许删除
	if execution.Status == "running" {
		tx.Rollback()
		return fmt.Errorf("无法删除正在运行的执行记录")
	}

	// 删除所有步骤执行记录
	if err := tx.Where("execution_id = ?", id).Delete(&model.WorkflowStepExecution{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除步骤执行记录失败: %w", err)
	}

	// 删除执行记录
	if err := tx.Where("id = ?", id).Delete(&model.WorkflowExecution{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除执行记录失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}
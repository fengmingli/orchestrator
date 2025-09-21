package service

import (
	"fmt"

	"gorm.io/gorm"
)

// ValidationService 数据一致性验证服务
type ValidationService struct {
	db *gorm.DB
}

// NewValidationService 创建验证服务
func NewValidationService(db *gorm.DB) *ValidationService {
	return &ValidationService{db: db}
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid bool     `json:"is_valid"`
	Errors  []string `json:"errors,omitempty"`
}

// ValidateDataConsistency 验证数据一致性
func (s *ValidationService) ValidateDataConsistency() (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid: true,
		Errors:  make([]string, 0),
	}

	// 验证模板步骤的完整性
	if err := s.validateTemplateSteps(result); err != nil {
		return nil, fmt.Errorf("验证模板步骤失败: %w", err)
	}

	// 验证执行记录的完整性
	if err := s.validateExecutions(result); err != nil {
		return nil, fmt.Errorf("验证执行记录失败: %w", err)
	}

	// 验证步骤执行记录的完整性
	if err := s.validateStepExecutions(result); err != nil {
		return nil, fmt.Errorf("验证步骤执行记录失败: %w", err)
	}

	return result, nil
}

// validateTemplateSteps 验证模板步骤的完整性
func (s *ValidationService) validateTemplateSteps(result *ValidationResult) error {
	// 检查模板步骤是否引用了不存在的步骤
	query := `
		SELECT ts.id, ts.template_id, ts.step_id 
		FROM workflow_template_steps ts 
		LEFT JOIN steps s ON ts.step_id = s.id 
		WHERE s.id IS NULL
	`
	
	type OrphanTemplateStep struct {
		ID         string `json:"id"`
		TemplateID string `json:"template_id"`
		StepID     string `json:"step_id"`
	}
	
	var orphanSteps []OrphanTemplateStep
	if err := s.db.Raw(query).Scan(&orphanSteps).Error; err != nil {
		return fmt.Errorf("查询孤立模板步骤失败: %w", err)
	}

	for _, orphan := range orphanSteps {
		result.IsValid = false
		result.Errors = append(result.Errors, 
			fmt.Sprintf("模板步骤 %s 引用了不存在的步骤 %s", orphan.ID, orphan.StepID))
	}

	// 检查模板步骤是否引用了不存在的模板
	query = `
		SELECT ts.id, ts.template_id, ts.step_id 
		FROM workflow_template_steps ts 
		LEFT JOIN workflow_templates t ON ts.template_id = t.id 
		WHERE t.id IS NULL
	`
	
	var orphanTemplateSteps []OrphanTemplateStep
	if err := s.db.Raw(query).Scan(&orphanTemplateSteps).Error; err != nil {
		return fmt.Errorf("查询孤立模板步骤失败: %w", err)
	}

	for _, orphan := range orphanTemplateSteps {
		result.IsValid = false
		result.Errors = append(result.Errors, 
			fmt.Sprintf("模板步骤 %s 引用了不存在的模板 %s", orphan.ID, orphan.TemplateID))
	}

	return nil
}

// validateExecutions 验证执行记录的完整性
func (s *ValidationService) validateExecutions(result *ValidationResult) error {
	// 检查执行记录是否引用了不存在的模板
	query := `
		SELECT e.id, e.template_id 
		FROM workflow_executions e 
		LEFT JOIN workflow_templates t ON e.template_id = t.id 
		WHERE t.id IS NULL
	`
	
	type OrphanExecution struct {
		ID         string `json:"id"`
		TemplateID string `json:"template_id"`
	}
	
	var orphanExecutions []OrphanExecution
	if err := s.db.Raw(query).Scan(&orphanExecutions).Error; err != nil {
		return fmt.Errorf("查询孤立执行记录失败: %w", err)
	}

	for _, orphan := range orphanExecutions {
		result.IsValid = false
		result.Errors = append(result.Errors, 
			fmt.Sprintf("执行记录 %s 引用了不存在的模板 %s", orphan.ID, orphan.TemplateID))
	}

	return nil
}

// validateStepExecutions 验证步骤执行记录的完整性
func (s *ValidationService) validateStepExecutions(result *ValidationResult) error {
	// 检查步骤执行记录是否引用了不存在的执行
	query := `
		SELECT se.id, se.execution_id, se.step_id 
		FROM workflow_step_executions se 
		LEFT JOIN workflow_executions e ON se.execution_id = e.id 
		WHERE e.id IS NULL
	`
	
	type OrphanStepExecution struct {
		ID          string `json:"id"`
		ExecutionID string `json:"execution_id"`
		StepID      string `json:"step_id"`
	}
	
	var orphanStepExecutions []OrphanStepExecution
	if err := s.db.Raw(query).Scan(&orphanStepExecutions).Error; err != nil {
		return fmt.Errorf("查询孤立步骤执行记录失败: %w", err)
	}

	for _, orphan := range orphanStepExecutions {
		result.IsValid = false
		result.Errors = append(result.Errors, 
			fmt.Sprintf("步骤执行记录 %s 引用了不存在的执行 %s", orphan.ID, orphan.ExecutionID))
	}

	// 检查步骤执行记录是否引用了不存在的步骤
	query = `
		SELECT se.id, se.execution_id, se.step_id 
		FROM workflow_step_executions se 
		LEFT JOIN steps s ON se.step_id = s.id 
		WHERE s.id IS NULL
	`
	
	var orphanStepExecutions2 []OrphanStepExecution
	if err := s.db.Raw(query).Scan(&orphanStepExecutions2).Error; err != nil {
		return fmt.Errorf("查询孤立步骤执行记录失败: %w", err)
	}

	for _, orphan := range orphanStepExecutions2 {
		result.IsValid = false
		result.Errors = append(result.Errors, 
			fmt.Sprintf("步骤执行记录 %s 引用了不存在的步骤 %s", orphan.ID, orphan.StepID))
	}

	return nil
}

// CleanupOrphanedData 清理孤立数据（谨慎使用）
func (s *ValidationService) CleanupOrphanedData() error {
	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除引用不存在步骤的模板步骤
	if err := tx.Exec(`
		DELETE FROM workflow_template_steps 
		WHERE step_id NOT IN (SELECT id FROM steps)
	`).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除孤立模板步骤失败: %w", err)
	}

	// 删除引用不存在模板的模板步骤
	if err := tx.Exec(`
		DELETE FROM workflow_template_steps 
		WHERE template_id NOT IN (SELECT id FROM workflow_templates)
	`).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除孤立模板步骤失败: %w", err)
	}

	// 删除引用不存在模板的执行记录（及其步骤执行记录）
	if err := tx.Exec(`
		DELETE FROM workflow_step_executions 
		WHERE execution_id IN (
			SELECT e.id FROM workflow_executions e 
			LEFT JOIN workflow_templates t ON e.template_id = t.id 
			WHERE t.id IS NULL
		)
	`).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除孤立步骤执行记录失败: %w", err)
	}

	if err := tx.Exec(`
		DELETE FROM workflow_executions 
		WHERE template_id NOT IN (SELECT id FROM workflow_templates)
	`).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除孤立执行记录失败: %w", err)
	}

	// 删除引用不存在执行的步骤执行记录
	if err := tx.Exec(`
		DELETE FROM workflow_step_executions 
		WHERE execution_id NOT IN (SELECT id FROM workflow_executions)
	`).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除孤立步骤执行记录失败: %w", err)
	}

	// 删除引用不存在步骤的步骤执行记录
	if err := tx.Exec(`
		DELETE FROM workflow_step_executions 
		WHERE step_id NOT IN (SELECT id FROM steps)
	`).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除孤立步骤执行记录失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}
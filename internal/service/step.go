package service

import (
	"encoding/json"
	"fmt"

	"github.com/fengmingli/orchestrator/internal/model"
	"gorm.io/gorm"
)

// StepService 步骤服务
type StepService struct {
	db *gorm.DB
}

// NewStepService 创建步骤服务
func NewStepService(db *gorm.DB) *StepService {
	return &StepService{db: db}
}

// CreateStep 创建步骤
func (s *StepService) CreateStep(req *model.StepCreateRequest) (*model.Step, error) {
	step := &model.Step{
		Name:          req.Name,
		Description:   req.Description,
		Parameters:    req.Parameters,
		CreatorEmail:  req.CreatorEmail,
		ExceptionType: req.ExceptionType,
		ExecutorType:  req.ExecutorType,
		Timeout:       req.Timeout,
		Retries:       req.Retries,
	}

	// 根据执行器类型设置对应字段
	switch req.ExecutorType {
	case "http":
		step.HTTPMethod = req.HTTPMethod
		step.HTTPURL = req.HTTPURL
		step.HTTPHeaders = req.HTTPHeaders
		step.HTTPBody = req.HTTPBody
	case "shell":
		step.ShellScript = req.ShellScript
		step.ShellEnv = req.ShellEnv
	}

	// 验证参数
	if err := s.validateStepConfig(step); err != nil {
		return nil, fmt.Errorf("参数验证失败: %w", err)
	}

	if err := s.db.Create(step).Error; err != nil {
		return nil, fmt.Errorf("创建步骤失败: %w", err)
	}

	return step, nil
}

// ListSteps 获取步骤列表
func (s *StepService) ListSteps(page, size int, executorType, creatorEmail string) ([]*model.Step, int64, error) {
	var steps []*model.Step
	var total int64

	query := s.db.Model(&model.Step{})

	// 筛选条件
	if executorType != "" {
		query = query.Where("executor_type = ?", executorType)
	}
	if creatorEmail != "" {
		query = query.Where("creator_email = ?", creatorEmail)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取步骤总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at desc").Find(&steps).Error; err != nil {
		return nil, 0, fmt.Errorf("获取步骤列表失败: %w", err)
	}

	return steps, total, nil
}

// GetStep 获取步骤详情
func (s *StepService) GetStep(id string) (*model.Step, error) {
	var step model.Step
	if err := s.db.Where("id = ?", id).First(&step).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("步骤不存在")
		}
		return nil, fmt.Errorf("获取步骤失败: %w", err)
	}

	return &step, nil
}

// UpdateStep 更新步骤
func (s *StepService) UpdateStep(id string, req *model.StepUpdateRequest) (*model.Step, error) {
	var step model.Step
	if err := s.db.Where("id = ?", id).First(&step).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("步骤不存在")
		}
		return nil, fmt.Errorf("获取步骤失败: %w", err)
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Parameters != nil {
		updates["parameters"] = *req.Parameters
	}
	if req.ExceptionType != nil {
		updates["exception_type"] = *req.ExceptionType
	}
	if req.ExecutorType != nil {
		updates["executor_type"] = *req.ExecutorType
	}
	if req.Timeout != nil {
		updates["timeout"] = *req.Timeout
	}
	if req.Retries != nil {
		updates["retries"] = *req.Retries
	}

	// HTTP相关字段
	if req.HTTPMethod != nil {
		updates["http_method"] = *req.HTTPMethod
	}
	if req.HTTPURL != nil {
		updates["http_url"] = *req.HTTPURL
	}
	if req.HTTPHeaders != nil {
		updates["http_headers"] = *req.HTTPHeaders
	}
	if req.HTTPBody != nil {
		updates["http_body"] = *req.HTTPBody
	}

	// Shell相关字段
	if req.ShellScript != nil {
		updates["shell_script"] = *req.ShellScript
	}
	if req.ShellEnv != nil {
		updates["shell_env"] = *req.ShellEnv
	}

	if err := s.db.Model(&step).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新步骤失败: %w", err)
	}

	return &step, nil
}

// DeleteStep 删除步骤
func (s *StepService) DeleteStep(id string) error {
	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查步骤是否存在
	var step model.Step
	if err := tx.Where("id = ?", id).First(&step).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("步骤不存在")
		}
		return fmt.Errorf("获取步骤失败: %w", err)
	}

	// 检查步骤是否被模板使用
	var templateStepCount int64
	if err := tx.Model(&model.WorkflowTemplateStep{}).Where("step_id = ?", id).Count(&templateStepCount).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("检查步骤使用情况失败: %w", err)
	}

	if templateStepCount > 0 {
		tx.Rollback()
		return fmt.Errorf("步骤正在被 %d 个模板使用，无法删除", templateStepCount)
	}

	// 检查步骤是否被执行记录使用
	var stepExecutionCount int64
	if err := tx.Model(&model.WorkflowStepExecution{}).Where("step_id = ?", id).Count(&stepExecutionCount).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("检查步骤执行记录失败: %w", err)
	}

	if stepExecutionCount > 0 {
		tx.Rollback()
		return fmt.Errorf("步骤存在 %d 个执行记录，无法删除", stepExecutionCount)
	}

	// 删除步骤
	if err := tx.Where("id = ?", id).Delete(&model.Step{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除步骤失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// ValidateStep 验证步骤参数
func (s *StepService) ValidateStep(id string) (bool, error) {
	step, err := s.GetStep(id)
	if err != nil {
		return false, err
	}

	if err := s.validateStepConfig(step); err != nil {
		return false, nil
	}

	return true, nil
}

// validateStepConfig 验证步骤配置
func (s *StepService) validateStepConfig(step *model.Step) error {
	switch step.ExecutorType {
	case "http":
		if step.HTTPURL == "" {
			return fmt.Errorf("HTTP任务必须提供URL")
		}
		if step.HTTPMethod == "" {
			step.HTTPMethod = "GET" // 默认GET方法
		}
		// 验证JSON格式
		if step.HTTPHeaders != "" {
			var headers map[string]string
			if err := json.Unmarshal([]byte(step.HTTPHeaders), &headers); err != nil {
				return fmt.Errorf("HTTP请求头格式不正确: %w", err)
			}
		}

	case "shell":
		if step.ShellScript == "" {
			return fmt.Errorf("Shell任务必须提供脚本")
		}
		// 验证环境变量JSON格式
		if step.ShellEnv != "" {
			var env map[string]string
			if err := json.Unmarshal([]byte(step.ShellEnv), &env); err != nil {
				return fmt.Errorf("环境变量格式不正确: %w", err)
			}
		}

	case "func":
		// 函数任务的参数在运行时验证
		break

	default:
		return fmt.Errorf("不支持的执行器类型: %s", step.ExecutorType)
	}

	return nil
}
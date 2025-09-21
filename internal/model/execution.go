package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkflowExecution 工作流执行记录
type WorkflowExecution struct {
	ID         string     `json:"id" gorm:"primaryKey;type:char(36)"`
	TemplateID string     `json:"template_id" gorm:"not null;comment:模板ID"`
	Status     string     `json:"status" gorm:"default:pending;comment:执行状态 pending/running/success/failed/cancelled"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Duration   int64      `json:"duration" gorm:"comment:执行时长(毫秒)"`
	Error      string     `json:"error,omitempty" gorm:"type:text;comment:错误信息"`
	CreatedBy  string     `json:"created_by" gorm:"comment:执行者"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// 保留关联字段但移除外键约束
	Template *WorkflowTemplate       `json:"template" gorm:"-"`
	Steps    []WorkflowStepExecution `json:"steps" gorm:"-"`
}

// WorkflowStepExecution 工作流步骤执行记录
type WorkflowStepExecution struct {
	ID          string     `json:"id" gorm:"primaryKey;type:char(36)"`
	ExecutionID string     `json:"execution_id" gorm:"not null;comment:执行ID"`
	StepID      string     `json:"step_id" gorm:"not null;comment:步骤ID"`
	Status      string     `json:"status" gorm:"default:pending;comment:执行状态"`
	Output      string     `json:"output" gorm:"type:text;comment:输出结果"`
	Error       string     `json:"error,omitempty" gorm:"type:text;comment:错误信息"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	Duration    int64      `json:"duration" gorm:"comment:执行时长(毫秒)"`
	RetryCount  int        `json:"retry_count" gorm:"default:0;comment:重试次数"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// 保留关联字段但移除外键约束
	Step *Step `json:"step" gorm:"-"`
}

// BeforeCreate GORM钩子，创建前生成UUID
func (we *WorkflowExecution) BeforeCreate(tx *gorm.DB) (err error) {
	if we.ID == "" {
		we.ID = uuid.New().String()
	}
	return
}

func (wse *WorkflowStepExecution) BeforeCreate(tx *gorm.DB) (err error) {
	if wse.ID == "" {
		wse.ID = uuid.New().String()
	}
	return
}

// TableName 指定表名
func (WorkflowExecution) TableName() string {
	return "workflow_executions"
}

func (WorkflowStepExecution) TableName() string {
	return "workflow_step_executions"
}

// ExecutionCreateRequest 创建执行请求
type ExecutionCreateRequest struct {
	TemplateID string `json:"template_id" binding:"required"`
	CreatedBy  string `json:"created_by"`
}

// ExecutionStatusUpdate 执行状态更新
type ExecutionStatusUpdate struct {
	Status     string     `json:"status" binding:"required,oneof=pending running success failed cancelled"`
	Error      string     `json:"error,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// StepExecutionStatusUpdate 步骤执行状态更新
type StepExecutionStatusUpdate struct {
	Status     string     `json:"status" binding:"required,oneof=pending running success failed skipped"`
	Output     string     `json:"output,omitempty"`
	Error      string     `json:"error,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	RetryCount int        `json:"retry_count,omitempty"`
}

// ExecutionSummary 执行摘要
type ExecutionSummary struct {
	ID           string     `json:"id"`
	TemplateID   string     `json:"template_id"`
	TemplateName string     `json:"template_name"`
	Status       string     `json:"status"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Duration     int64      `json:"duration"`
	TotalSteps   int        `json:"total_steps"`
	SuccessSteps int        `json:"success_steps"`
	FailedSteps  int        `json:"failed_steps"`
	CreatedBy    string     `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
}

package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkflowTemplate 工作流模板
type WorkflowTemplate struct {
	ID           string    `json:"id" gorm:"primaryKey;type:char(36)"`
	Name         string    `json:"name" gorm:"not null;comment:模板名称"`
	Description  string    `json:"description" gorm:"comment:模板描述"`
	CreatorEmail string    `json:"creator_email" gorm:"not null;comment:创建者邮箱"`
	IsActive     bool      `json:"is_active" gorm:"default:true;comment:是否激活"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// 保留关联字段但移除外键约束
	Steps []WorkflowTemplateStep `json:"steps" gorm:"-"`
}

// WorkflowTemplateStep 模板步骤关联
type WorkflowTemplateStep struct {
	ID           string    `json:"id" gorm:"primaryKey;type:char(36)"`
	TemplateID   string    `json:"template_id" gorm:"not null;comment:模板ID"`
	StepID       string    `json:"step_id" gorm:"not null;comment:步骤ID"`
	Dependencies string    `json:"dependencies" gorm:"type:text;comment:依赖关系JSON"`
	RunMode      string    `json:"run_mode" gorm:"default:serial;comment:运行模式 serial/parallel"`
	OnFailure    string    `json:"on_failure" gorm:"default:abort;comment:失败策略 abort/skip"`
	Order        int       `json:"order" gorm:"comment:步骤顺序"`
	CreatedAt    time.Time `json:"created_at"`

	// 保留关联字段但移除外键约束
	Step *Step `json:"step" gorm:"-"`
}

// BeforeCreate GORM钩子，创建前生成UUID
func (wt *WorkflowTemplate) BeforeCreate(tx *gorm.DB) (err error) {
	if wt.ID == "" {
		wt.ID = uuid.New().String()
	}
	return
}

func (wts *WorkflowTemplateStep) BeforeCreate(tx *gorm.DB) (err error) {
	if wts.ID == "" {
		wts.ID = uuid.New().String()
	}
	return
}

// TableName 指定表名
func (WorkflowTemplate) TableName() string {
	return "workflow_templates"
}

func (WorkflowTemplateStep) TableName() string {
	return "workflow_template_steps"
}

// TemplateCreateRequest 创建模板请求
type TemplateCreateRequest struct {
	Name         string                      `json:"name" binding:"required"`
	Description  string                      `json:"description"`
	CreatorEmail string                      `json:"creator_email" binding:"required,email"`
	Steps        []TemplateStepCreateRequest `json:"steps" binding:"required,min=1"`
}

// TemplateStepCreateRequest 模板步骤创建请求
type TemplateStepCreateRequest struct {
	StepID       string   `json:"step_id" binding:"required"`
	Dependencies []string `json:"dependencies"`
	RunMode      string   `json:"run_mode" binding:"oneof=serial parallel"`
	OnFailure    string   `json:"on_failure" binding:"oneof=abort skip"`
	Order        int      `json:"order"`
}

// TemplateUpdateRequest 更新模板请求
type TemplateUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// DAGNode DAG节点定义（用于前端可视化）
type DAGNode struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Dependencies []string               `json:"dependencies"`
	RunMode      string                 `json:"run_mode"`
	OnFailure    string                 `json:"on_failure"`
	Position     map[string]interface{} `json:"position,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
}

// DAGDefinition DAG定义
type DAGDefinition struct {
	Nodes []DAGNode `json:"nodes"`
	Edges []DAGEdge `json:"edges"`
}

// DAGEdge DAG边定义
type DAGEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type,omitempty"`
}


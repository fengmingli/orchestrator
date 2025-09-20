package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Step 步骤模型
type Step struct {
	ID            string `json:"id" gorm:"primaryKey;type:char(36)"`
	Name          string `json:"name" gorm:"not null;comment:步骤名称"`
	Description   string `json:"description" gorm:"comment:步骤描述"`
	Parameters    string `json:"parameters" gorm:"type:text;comment:参数JSON"`
	CreatorEmail  string `json:"creator_email" gorm:"not null;comment:创建者邮箱"`
	ExceptionType string `json:"exception_type" gorm:"comment:异常类型"`
	ExecutorType  string `json:"executor_type" gorm:"not null;comment:执行器类型 http/shell/func"`

	// HTTP相关字段
	HTTPMethod  string `json:"http_method,omitempty" gorm:"comment:HTTP方法"`
	HTTPURL     string `json:"http_url,omitempty" gorm:"comment:HTTP URL"`
	HTTPHeaders string `json:"http_headers,omitempty" gorm:"type:text;comment:HTTP请求头JSON"`
	HTTPBody    string `json:"http_body,omitempty" gorm:"type:text;comment:HTTP请求体"`

	// Shell相关字段
	ShellScript string `json:"shell_script,omitempty" gorm:"type:text;comment:Shell脚本"`
	ShellEnv    string `json:"shell_env,omitempty" gorm:"type:text;comment:环境变量JSON"`

	// 通用字段
	Timeout   int       `json:"timeout" gorm:"default:30;comment:超时时间(秒)"`
	Retries   int       `json:"retries" gorm:"default:3;comment:重试次数"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate GORM钩子，创建前生成UUID
func (s *Step) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return
}

// TableName 指定表名
func (Step) TableName() string {
	return "steps"
}

// StepCreateRequest 创建步骤请求
type StepCreateRequest struct {
	Name          string `json:"name" binding:"required,min=1,max=128,alphanum_underscore_dash"`
	Description   string `json:"description"`
	Parameters    string `json:"parameters"`
	CreatorEmail  string `json:"creator_email" binding:"required,email"`
	ExceptionType string `json:"exception_type"`
	ExecutorType  string `json:"executor_type" binding:"required,oneof=http shell func"`

	// HTTP相关
	HTTPMethod  string `json:"http_method,omitempty"`
	HTTPURL     string `json:"http_url,omitempty"`
	HTTPHeaders string `json:"http_headers,omitempty"`
	HTTPBody    string `json:"http_body,omitempty"`

	// Shell相关
	ShellScript string `json:"shell_script,omitempty"`
	ShellEnv    string `json:"shell_env,omitempty"`

	// 通用
	Timeout int `json:"timeout"`
	Retries int `json:"retries"`
}

// StepUpdateRequest 更新步骤请求
type StepUpdateRequest struct {
	Name          *string `json:"name,omitempty" binding:"omitempty,min=1,max=128,alphanum_underscore_dash"`
	Description   *string `json:"description,omitempty"`
	Parameters    *string `json:"parameters,omitempty"`
	ExceptionType *string `json:"exception_type,omitempty"`
	ExecutorType  *string `json:"executor_type,omitempty"`

	// HTTP相关
	HTTPMethod  *string `json:"http_method,omitempty"`
	HTTPURL     *string `json:"http_url,omitempty"`
	HTTPHeaders *string `json:"http_headers,omitempty"`
	HTTPBody    *string `json:"http_body,omitempty"`

	// Shell相关
	ShellScript *string `json:"shell_script,omitempty"`
	ShellEnv    *string `json:"shell_env,omitempty"`

	// 通用
	Timeout *int `json:"timeout,omitempty"`
	Retries *int `json:"retries,omitempty"`
}

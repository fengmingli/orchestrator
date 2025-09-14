package model

import "time"

/**
 * @Author: LFM
 * @Date: 2025/9/14 12:00
 * @Since: 1.0.0
 * @Desc: TODO
 */

// Template 预案模板
type Template struct {
	ID          string `gorm:"primaryKey;size:64"`
	Name        string `gorm:"size:128"`
	Description string
	CreatedAt   time.Time
	Steps       []TemplateStep `gorm:"foreignKey:TemplateID;references:ID"`
}

// TemplateStep 模板步骤
type TemplateStep struct {
	ID           string         `gorm:"primaryKey;size:64"`
	TemplateID   string         `gorm:"size:64;index"`
	StepKey      string         `gorm:"size:64"`
	Name         string         `gorm:"size:128"`
	Type         string         `gorm:"size:32"` // http / shell / func
	Parameters   datatypes.JSON // 具体参数
	Mode         string         `gorm:"size:16;default:serial"` // serial / parallel
	Dependencies datatypes.JSON // []string
	TimeoutSec   int            `gorm:"default:30"`
	RetryTimes   int            `gorm:"default:0"`
}

// Execution 一次执行实例
type Execution struct {
	ID         string `gorm:"primaryKey;size:64"`
	TemplateID string `gorm:"size:64;index"`
	Status     string `gorm:"size:16;default:pending"` // pending / running / success / failed
	StartedAt  *time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
}

// StepExecution 步骤执行记录
type StepExecution struct {
	ID          string `gorm:"primaryKey;size:64"`
	ExecutionID string `gorm:"size:64;index"`
	StepID      string `gorm:"size:64;index"`
	Status      string `gorm:"size:16;default:pending"`
	Output      string `gorm:"type:text"`
	Error       string `gorm:"type:text"`
	StartedAt   *time.Time
	FinishedAt  *time.Time
	CreatedAt   time.Time
}

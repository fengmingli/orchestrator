package task

import (
	"context"
	"fmt"
	"time"
)

// Task 定义了所有任务必须实现的接口
type Task interface {
	// 基本属性
	GetID() string
	GetName() string
	GetType() TaskType

	// 执行相关
	Execute(ctx context.Context) (ExecResult, error)
	Validate() error

	// 超时控制
	GetTimeout() time.Duration
}

// TaskType 任务类型枚举
type TaskType string

const (
	TaskTypeHTTP  TaskType = "http"
	TaskTypeShell TaskType = "shell"
	TaskTypeFunc  TaskType = "func"
)

// ExecStatus 执行状态
type ExecStatus int

const (
	ExecStatusPending ExecStatus = iota
	ExecStatusRunning
	ExecStatusWithSuccess
	ExecStatusWithFailed
	ExecStatusSkipped
)

// ExecResult 执行结果
type ExecResult struct {
	TaskID     string                 `json:"task_id"`
	Output     interface{}            `json:"output"`
	Error      string                 `json:"error,omitempty"`
	Duration   time.Duration          `json:"duration"`
	StartTime  time.Time              `json:"start_time"`
	FinishTime time.Time              `json:"finish_time"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// BaseTask 基础任务实现，其他任务可以嵌入此结构
type BaseTask struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Type    TaskType      `json:"type"`
	Timeout time.Duration `json:"timeout"`
}

func (t *BaseTask) GetID() string {
	return t.ID
}

func (t *BaseTask) GetName() string {
	return t.Name
}

func (t *BaseTask) GetType() TaskType {
	return t.Type
}

func (t *BaseTask) GetTimeout() time.Duration {
	if t.Timeout <= 0 {
		return 30 * time.Second // 默认30秒
	}
	return t.Timeout
}

func (t *BaseTask) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if t.Name == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	return nil
}

package task

import (
	"context"
	"fmt"
	"time"
)

// FuncTask 函数任务实现 - 执行Go函数
type FuncTask struct {
	BaseTask
	Func     func(ctx context.Context) (interface{}, error) `json:"-"` // 不序列化函数
	Metadata map[string]interface{}                         `json:"metadata"`
}

// NewFuncTask 创建新的函数任务
func NewFuncTask(id, name string, fn func(ctx context.Context) (interface{}, error)) *FuncTask {
	return &FuncTask{
		BaseTask: BaseTask{
			ID:      id,
			Name:    name,
			Type:    TaskTypeFunc,
			Timeout: 30 * time.Second,
		},
		Func:     fn,
		Metadata: make(map[string]interface{}),
	}
}

// WithMetadata 设置元数据
func (f *FuncTask) WithMetadata(metadata map[string]interface{}) *FuncTask {
	f.Metadata = metadata
	return f
}

// AddMetadata 添加单个元数据
func (f *FuncTask) AddMetadata(key string, value interface{}) *FuncTask {
	if f.Metadata == nil {
		f.Metadata = make(map[string]interface{})
	}
	f.Metadata[key] = value
	return f
}

// Validate 验证函数任务参数
func (f *FuncTask) Validate() error {
	if err := f.BaseTask.Validate(); err != nil {
		return err
	}

	if f.Func == nil {
		return fmt.Errorf("function cannot be nil")
	}

	return nil
}

// Execute 执行函数任务
func (f *FuncTask) Execute(ctx context.Context) (ExecResult, error) {
	start := time.Now()
	result := ExecResult{
		TaskID:    f.ID,
		StartTime: start,
		Metadata:  make(map[string]interface{}),
	}

	// 复制元数据
	for k, v := range f.Metadata {
		result.Metadata[k] = v
	}

	// 执行函数
	output, err := f.Func(ctx)

	// 填充结果
	result.FinishTime = time.Now()
	result.Duration = result.FinishTime.Sub(start)
	result.Output = output

	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	return result, nil
}

// SimpleFunc 简单函数类型定义
type SimpleFunc func() (interface{}, error)

// NewSimpleFuncTask 创建简单函数任务（不需要context）
func NewSimpleFuncTask(id, name string, fn SimpleFunc) *FuncTask {
	return NewFuncTask(id, name, func(ctx context.Context) (interface{}, error) {
		return fn()
	})
}

// VoidFunc 无返回值函数类型定义
type VoidFunc func() error

// NewVoidFuncTask 创建无返回值函数任务
func NewVoidFuncTask(id, name string, fn VoidFunc) *FuncTask {
	return NewFuncTask(id, name, func(ctx context.Context) (interface{}, error) {
		err := fn()
		return nil, err
	})
}

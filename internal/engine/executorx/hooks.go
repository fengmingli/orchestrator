package executorx

import (
	"context"
	"time"

	"github.com/fengmingli/orchestrator/internal/engine/task"
	"github.com/sirupsen/logrus"
)

// LoggingHook 日志Hook实现
type LoggingHook struct {
	logger *logrus.Entry
}

// NewLoggingHook 创建新的日志Hook
func NewLoggingHook(logger *logrus.Entry) *LoggingHook {
	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}
	return &LoggingHook{
		logger: logger,
	}
}

func (h *LoggingHook) OnRunning(ctx context.Context, taskInst task.Task, result WarpExecResult) {
	h.logger.WithFields(logrus.Fields{
		"task_id":   taskInst.GetID(),
		"task_name": taskInst.GetName(),
		"task_type": taskInst.GetType(),
	}).Info("任务开始执行")
}

func (h *LoggingHook) OnSuccess(ctx context.Context, taskInst task.Task, result WarpExecResult) {
	h.logger.WithFields(logrus.Fields{
		"task_id":   taskInst.GetID(),
		"task_name": taskInst.GetName(),
		"task_type": taskInst.GetType(),
		"duration":  result.Duration,
	}).Info("任务执行成功")
}

func (h *LoggingHook) OnFailure(ctx context.Context, taskInst task.Task, result WarpExecResult) {
	h.logger.WithFields(logrus.Fields{
		"task_id":        taskInst.GetID(),
		"task_name":      taskInst.GetName(),
		"task_type":      taskInst.GetType(),
		"error":          result.Err,
		"failure_reason": result.FailureReason,
	}).Error("任务执行失败")
}

// MetricsHook 指标收集Hook接口
type MetricsHook struct {
	taskStartTimes map[string]time.Time
}

// NewMetricsHook 创建新的指标Hook
func NewMetricsHook() *MetricsHook {
	return &MetricsHook{
		taskStartTimes: make(map[string]time.Time),
	}
}

func (m *MetricsHook) OnRunning(ctx context.Context, taskInst task.Task, result WarpExecResult) {
	m.taskStartTimes[taskInst.GetID()] = time.Now()
	// 这里可以发送任务开始的指标
}

func (m *MetricsHook) OnSuccess(ctx context.Context, taskInst task.Task, result WarpExecResult) {
	if startTime, exists := m.taskStartTimes[taskInst.GetID()]; exists {
		duration := time.Since(startTime)
		// 这里可以发送任务成功的指标，包含执行时间
		_ = duration
		delete(m.taskStartTimes, taskInst.GetID())
	}
}

func (m *MetricsHook) OnFailure(ctx context.Context, taskInst task.Task, result WarpExecResult) {
	if startTime, exists := m.taskStartTimes[taskInst.GetID()]; exists {
		duration := time.Since(startTime)
		// 这里可以发送任务失败的指标，包含执行时间
		_ = duration
		delete(m.taskStartTimes, taskInst.GetID())
	}
}

// CompositeHook 组合Hook，支持多个Hook
type CompositeHook struct {
	hooks []Hook
}

// NewCompositeHook 创建组合Hook
func NewCompositeHook(hooks ...Hook) *CompositeHook {
	return &CompositeHook{
		hooks: hooks,
	}
}

// AddHook 添加Hook
func (c *CompositeHook) AddHook(hook Hook) {
	if hook != nil {
		c.hooks = append(c.hooks, hook)
	}
}

func (c *CompositeHook) OnRunning(ctx context.Context, taskInst task.Task, result WarpExecResult) {
	for _, hook := range c.hooks {
		hook.OnRunning(ctx, taskInst, result)
	}
}

func (c *CompositeHook) OnSuccess(ctx context.Context, taskInst task.Task, result WarpExecResult) {
	for _, hook := range c.hooks {
		hook.OnSuccess(ctx, taskInst, result)
	}
}

func (c *CompositeHook) OnFailure(ctx context.Context, taskInst task.Task, result WarpExecResult) {
	for _, hook := range c.hooks {
		hook.OnFailure(ctx, taskInst, result)
	}
}

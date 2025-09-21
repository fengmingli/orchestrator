package lock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DefaultWorkflowLockProvider 默认工作流锁提供者实现
type DefaultWorkflowLockProvider struct {
	lockProvider LockProvider
	instanceID   string
	logger       *logrus.Entry
	config       *LockConfig
}

// NewWorkflowLockProvider 创建工作流锁提供者
func NewWorkflowLockProvider(lockProvider LockProvider, config *LockConfig) WorkflowLockProvider {
	if config == nil {
		config = DefaultLockConfig()
	}

	// 生成实例ID，用于标识当前副本
	instanceID := generateInstanceID()

	return &DefaultWorkflowLockProvider{
		lockProvider: lockProvider,
		instanceID:   instanceID,
		logger:       logrus.NewEntry(logrus.New()).WithField("component", "workflow_lock_provider"),
		config:       config,
	}
}

// LockWorkflowExecution 为工作流执行加锁
func (p *DefaultWorkflowLockProvider) LockWorkflowExecution(ctx context.Context, executionID string) (WorkflowLockHandle, error) {
	lockKey := fmt.Sprintf("workflow_execution:%s", executionID)

	opts := LockOptions{
		LockKey:    lockKey,
		Owner:      p.instanceID,
		TTL:        5 * time.Minute, // 工作流锁默认5分钟超时
		RetryCount: 0,               // 工作流执行不重试，避免重复执行
		RetryDelay: time.Second,
	}

	logger := p.logger.WithFields(logrus.Fields{
		"execution_id": executionID,
		"lock_key":     lockKey,
		"instance_id":  p.instanceID,
	})

	logger.Info("尝试获取工作流执行锁")

	if err := p.lockProvider.Lock(ctx, opts); err != nil {
		if errors.Is(err, ErrLockTimeout) || errors.Is(err, ErrLockAlreadyExists) {
			logger.Info("工作流已被其他副本执行，跳过")
			return nil, ErrWorkflowAlreadyRunning
		}
		logger.WithError(err).Error("获取工作流执行锁失败")
		return nil, err
	}

	// 返回锁句柄用于后续操作
	return &DefaultWorkflowLockHandle{
		provider:    p,
		lockKey:     lockKey,
		executionID: executionID,
		logger:      logger,
	}, nil
}

// LockWorkflowTemplate 为工作流模板加锁（防止同一模板的多个实例同时执行）
func (p *DefaultWorkflowLockProvider) LockWorkflowTemplate(ctx context.Context, templateID string) (WorkflowLockHandle, error) {
	lockKey := fmt.Sprintf("workflow_template:%s", templateID)

	opts := LockOptions{
		LockKey:    lockKey,
		Owner:      p.instanceID,
		TTL:        10 * time.Minute, // 模板锁默认10分钟超时
		RetryCount: 2,                // 模板锁可以重试
		RetryDelay: 5 * time.Second,
	}

	logger := p.logger.WithFields(logrus.Fields{
		"template_id": templateID,
		"lock_key":    lockKey,
		"instance_id": p.instanceID,
	})

	logger.Info("尝试获取工作流模板锁")

	if err := p.lockProvider.Lock(ctx, opts); err != nil {
		logger.WithError(err).Error("获取工作流模板锁失败")
		return nil, err
	}

	return &DefaultWorkflowLockHandle{
		provider:   p,
		lockKey:    lockKey,
		templateID: templateID,
		logger:     logger,
	}, nil
}

// Close 关闭工作流锁提供者
func (p *DefaultWorkflowLockProvider) Close() error {
	p.logger.Info("关闭工作流锁提供者")
	return p.lockProvider.Close()
}

// DefaultWorkflowLockHandle 默认工作流锁句柄实现
type DefaultWorkflowLockHandle struct {
	provider    *DefaultWorkflowLockProvider
	lockKey     string
	executionID string
	templateID  string
	logger      *logrus.Entry
}

// Unlock 释放锁
func (h *DefaultWorkflowLockHandle) Unlock(ctx context.Context) error {
	h.logger.Info("释放工作流锁")
	return h.provider.lockProvider.Unlock(ctx, h.lockKey, h.provider.instanceID)
}

// Refresh 刷新锁的过期时间
func (h *DefaultWorkflowLockHandle) Refresh(ctx context.Context, ttl time.Duration) error {
	h.logger.WithField("ttl", ttl).Info("刷新工作流锁")
	return h.provider.lockProvider.RefreshLock(ctx, h.lockKey, h.provider.instanceID, ttl)
}

// GetLockKey 获取锁键
func (h *DefaultWorkflowLockHandle) GetLockKey() string {
	return h.lockKey
}

// GetExecutionID 获取执行ID
func (h *DefaultWorkflowLockHandle) GetExecutionID() string {
	return h.executionID
}

// GetTemplateID 获取模板ID
func (h *DefaultWorkflowLockHandle) GetTemplateID() string {
	return h.templateID
}

// 生成实例ID
func generateInstanceID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	// 格式：hostname-uuid前8位
	return fmt.Sprintf("%s-%s", hostname, uuid.New().String()[:8])
}

package lock

import (
	"context"
	"time"
)

// LockOptions 锁选项
type LockOptions struct {
	LockKey    string        // 锁的唯一标识
	Owner      string        // 锁的持有者标识
	TTL        time.Duration // 锁的生存时间
	RetryCount int           // 重试次数
	RetryDelay time.Duration // 重试间隔
}

// LockProvider 分布式锁提供者接口
type LockProvider interface {
	// Lock 获取分布式锁
	Lock(ctx context.Context, opts LockOptions) error
	
	// Unlock 释放分布式锁
	Unlock(ctx context.Context, lockKey, owner string) error
	
	// RefreshLock 刷新锁的过期时间
	RefreshLock(ctx context.Context, lockKey, owner string, ttl time.Duration) error
	
	// IsLocked 检查锁是否存在且未过期
	IsLocked(ctx context.Context, lockKey string) (bool, string, error)
	
	// Close 关闭锁提供者，清理资源
	Close() error
}

// WorkflowLockProvider 工作流锁提供者接口
type WorkflowLockProvider interface {
	// LockWorkflowExecution 为工作流执行加锁
	LockWorkflowExecution(ctx context.Context, executionID string) (WorkflowLockHandle, error)
	
	// LockWorkflowTemplate 为工作流模板加锁
	LockWorkflowTemplate(ctx context.Context, templateID string) (WorkflowLockHandle, error)
	
	// Close 关闭工作流锁提供者
	Close() error
}

// WorkflowLockHandle 工作流锁句柄接口
type WorkflowLockHandle interface {
	// Unlock 释放锁
	Unlock(ctx context.Context) error
	
	// Refresh 刷新锁的过期时间
	Refresh(ctx context.Context, ttl time.Duration) error
	
	// GetLockKey 获取锁键
	GetLockKey() string
	
	// GetExecutionID 获取执行ID（如果是执行锁）
	GetExecutionID() string
	
	// GetTemplateID 获取模板ID（如果是模板锁）
	GetTemplateID() string
}

// LockProviderType 锁提供者类型
type LockProviderType string

const (
	// LockProviderMySQL MySQL分布式锁
	LockProviderMySQL LockProviderType = "mysql"
	
	// LockProviderRedis Redis分布式锁
	LockProviderRedis LockProviderType = "redis"
	
	// LockProviderEtcd etcd分布式锁
	LockProviderEtcd LockProviderType = "etcd"
	
	// LockProviderMemory 内存分布式锁（仅用于测试）
	LockProviderMemory LockProviderType = "memory"
)

// LockConfig 锁配置
type LockConfig struct {
	// Provider 锁提供者类型
	Provider LockProviderType `json:"provider" yaml:"provider"`
	
	// MySQL 配置（当Provider为mysql时使用）
	MySQL *MySQLConfig `json:"mysql,omitempty" yaml:"mysql,omitempty"`
	
	// Redis 配置（当Provider为redis时使用）
	Redis *RedisConfig `json:"redis,omitempty" yaml:"redis,omitempty"`
	
	// Etcd 配置（当Provider为etcd时使用）
	Etcd *EtcdConfig `json:"etcd,omitempty" yaml:"etcd,omitempty"`
	
	// DefaultTTL 默认锁过期时间
	DefaultTTL time.Duration `json:"default_ttl" yaml:"default_ttl"`
	
	// DefaultRetryCount 默认重试次数
	DefaultRetryCount int `json:"default_retry_count" yaml:"default_retry_count"`
	
	// DefaultRetryDelay 默认重试延迟
	DefaultRetryDelay time.Duration `json:"default_retry_delay" yaml:"default_retry_delay"`
}

// MySQLConfig MySQL锁配置
type MySQLConfig struct {
	// 可以复用现有的gorm.DB连接，或者单独配置
	TableName string `json:"table_name" yaml:"table_name"`
}

// RedisConfig Redis锁配置
type RedisConfig struct {
	Addr     string `json:"addr" yaml:"addr"`
	Password string `json:"password" yaml:"password"`
	DB       int    `json:"db" yaml:"db"`
}

// EtcdConfig Etcd锁配置
type EtcdConfig struct {
	Endpoints []string `json:"endpoints" yaml:"endpoints"`
	Username  string   `json:"username" yaml:"username"`
	Password  string   `json:"password" yaml:"password"`
}

// DefaultLockConfig 返回默认锁配置
func DefaultLockConfig() *LockConfig {
	return &LockConfig{
		Provider:          LockProviderMySQL,
		DefaultTTL:        5 * time.Minute,
		DefaultRetryCount: 3,
		DefaultRetryDelay: time.Second,
		MySQL: &MySQLConfig{
			TableName: "distributed_locks",
		},
	}
}
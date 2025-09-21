package lock

import (
	"fmt"

	"gorm.io/gorm"
)

// LockProviderFactory 锁提供者工厂
type LockProviderFactory struct {
	db *gorm.DB // MySQL连接，用于MySQL锁提供者
}

// NewLockProviderFactory 创建锁提供者工厂
func NewLockProviderFactory(db *gorm.DB) *LockProviderFactory {
	return &LockProviderFactory{
		db: db,
	}
}

// CreateLockProvider 根据配置创建锁提供者
func (f *LockProviderFactory) CreateLockProvider(config *LockConfig) (LockProvider, error) {
	if config == nil {
		config = DefaultLockConfig()
	}

	switch config.Provider {
	case LockProviderMySQL:
		return f.createMySQLProvider(config.MySQL)
	case LockProviderRedis:
		return f.createRedisProvider(config.Redis)
	case LockProviderEtcd:
		return f.createEtcdProvider(config.Etcd)
	case LockProviderMemory:
		return f.createMemoryProvider()
	default:
		return nil, fmt.Errorf("不支持的锁提供者类型: %s", config.Provider)
	}
}

// CreateWorkflowLockProvider 创建工作流锁提供者
func (f *LockProviderFactory) CreateWorkflowLockProvider(config *LockConfig) (WorkflowLockProvider, error) {
	lockProvider, err := f.CreateLockProvider(config)
	if err != nil {
		return nil, err
	}

	return NewWorkflowLockProvider(lockProvider, config), nil
}

// createMySQLProvider 创建MySQL锁提供者
func (f *LockProviderFactory) createMySQLProvider(config *MySQLConfig) (LockProvider, error) {
	if f.db == nil {
		return nil, fmt.Errorf("MySQL数据库连接未配置")
	}

	return NewMySQLLockProvider(f.db, config)
}

// createRedisProvider 创建Redis锁提供者
func (f *LockProviderFactory) createRedisProvider(config *RedisConfig) (LockProvider, error) {
	// TODO: 实现Redis锁提供者
	return nil, fmt.Errorf("Redis锁提供者尚未实现")
}

// createEtcdProvider 创建Etcd锁提供者
func (f *LockProviderFactory) createEtcdProvider(config *EtcdConfig) (LockProvider, error) {
	// TODO: 实现Etcd锁提供者
	return nil, fmt.Errorf("Etcd锁提供者尚未实现")
}

// createMemoryProvider 创建内存锁提供者（仅用于测试）
func (f *LockProviderFactory) createMemoryProvider() (LockProvider, error) {
	return NewMemoryLockProvider(), nil
}

// LockManager 锁管理器（单例模式）
type LockManager struct {
	factory          *LockProviderFactory
	workflowProvider WorkflowLockProvider
	config           *LockConfig
}

// NewLockManager 创建锁管理器
func NewLockManager(db *gorm.DB, config *LockConfig) (*LockManager, error) {
	if config == nil {
		config = DefaultLockConfig()
	}

	factory := NewLockProviderFactory(db)
	workflowProvider, err := factory.CreateWorkflowLockProvider(config)
	if err != nil {
		return nil, fmt.Errorf("创建工作流锁提供者失败: %w", err)
	}

	return &LockManager{
		factory:          factory,
		workflowProvider: workflowProvider,
		config:           config,
	}, nil
}

// GetWorkflowLockProvider 获取工作流锁提供者
func (m *LockManager) GetWorkflowLockProvider() WorkflowLockProvider {
	return m.workflowProvider
}

// CreateLockProvider 创建新的锁提供者
func (m *LockManager) CreateLockProvider(providerType LockProviderType) (LockProvider, error) {
	config := *m.config // 复制配置
	config.Provider = providerType
	return m.factory.CreateLockProvider(&config)
}

// Close 关闭锁管理器
func (m *LockManager) Close() error {
	return m.workflowProvider.Close()
}

// GetConfig 获取锁配置
func (m *LockManager) GetConfig() *LockConfig {
	return m.config
}
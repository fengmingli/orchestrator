package lock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockProviderInterface_MySQL(t *testing.T) {
	db := setupTestDB(t)
	factory := NewLockProviderFactory(db)

	config := &LockConfig{
		Provider: LockProviderMySQL,
		MySQL: &MySQLConfig{
			TableName: "test_locks",
		},
	}

	provider, err := factory.CreateLockProvider(config)
	require.NoError(t, err)
	defer provider.Close()

	testLockProviderInterface(t, provider)
}

func TestLockProviderInterface_Memory(t *testing.T) {
	factory := NewLockProviderFactory(nil)

	config := &LockConfig{
		Provider: LockProviderMemory,
	}

	provider, err := factory.CreateLockProvider(config)
	require.NoError(t, err)
	defer provider.Close()

	testLockProviderInterface(t, provider)
}

func testLockProviderInterface(t *testing.T, provider LockProvider) {
	ctx := context.Background()

	// 测试基本锁操作
	t.Run("BasicLockUnlock", func(t *testing.T) {
		opts := LockOptions{
			LockKey: "test_interface_lock",
			Owner:   "test_owner",
			TTL:     30 * time.Second,
		}

		// 获取锁
		err := provider.Lock(ctx, opts)
		assert.NoError(t, err)

		// 检查锁状态
		locked, owner, err := provider.IsLocked(ctx, "test_interface_lock")
		assert.NoError(t, err)
		assert.True(t, locked)
		assert.Equal(t, "test_owner", owner)

		// 释放锁
		err = provider.Unlock(ctx, "test_interface_lock", "test_owner")
		assert.NoError(t, err)

		// 检查锁已释放
		locked, _, err = provider.IsLocked(ctx, "test_interface_lock")
		assert.NoError(t, err)
		assert.False(t, locked)
	})

	// 测试并发锁
	t.Run("ConcurrentLock", func(t *testing.T) {
		opts1 := LockOptions{
			LockKey:    "test_concurrent_interface",
			Owner:      "owner1",
			TTL:        30 * time.Second,
			RetryCount: 0,
		}

		opts2 := LockOptions{
			LockKey:    "test_concurrent_interface",
			Owner:      "owner2",
			TTL:        30 * time.Second,
			RetryCount: 0,
		}

		// 第一个获取锁
		err := provider.Lock(ctx, opts1)
		assert.NoError(t, err)

		// 第二个尝试获取锁，应该失败
		err = provider.Lock(ctx, opts2)
		assert.True(t, errors.Is(err, ErrLockTimeout))

		// 释放锁
		err = provider.Unlock(ctx, "test_concurrent_interface", "owner1")
		assert.NoError(t, err)

		// 现在第二个应该能获取锁
		err = provider.Lock(ctx, opts2)
		assert.NoError(t, err)

		// 清理
		err = provider.Unlock(ctx, "test_concurrent_interface", "owner2")
		assert.NoError(t, err)
	})

	// 测试锁刷新
	t.Run("RefreshLock", func(t *testing.T) {
		opts := LockOptions{
			LockKey: "test_refresh_interface",
			Owner:   "test_owner",
			TTL:     time.Second,
		}

		// 获取锁
		err := provider.Lock(ctx, opts)
		assert.NoError(t, err)

		// 刷新锁
		err = provider.RefreshLock(ctx, "test_refresh_interface", "test_owner", 5*time.Minute)
		assert.NoError(t, err)

		// 清理
		err = provider.Unlock(ctx, "test_refresh_interface", "test_owner")
		assert.NoError(t, err)
	})
}

func TestWorkflowLockProviderInterface(t *testing.T) {
	db := setupTestDB(t)
	factory := NewLockProviderFactory(db)

	config := DefaultLockConfig()
	workflowProvider, err := factory.CreateWorkflowLockProvider(config)
	require.NoError(t, err)
	defer workflowProvider.Close()

	ctx := context.Background()

	// 测试工作流执行锁
	t.Run("WorkflowExecutionLock", func(t *testing.T) {
		lock, err := workflowProvider.LockWorkflowExecution(ctx, "test_execution_123")
		assert.NoError(t, err)
		assert.NotNil(t, lock)

		// 验证锁信息
		assert.Equal(t, "workflow_execution:test_execution_123", lock.GetLockKey())
		assert.Equal(t, "test_execution_123", lock.GetExecutionID())

		// 创建另一个工作流锁提供者模拟不同实例
		workflowProvider2, err := factory.CreateWorkflowLockProvider(config)
		require.NoError(t, err)
		defer workflowProvider2.Close()

		// 尝试用不同实例获取同一个锁，应该失败
		lock2, err := workflowProvider2.LockWorkflowExecution(ctx, "test_execution_123")
		assert.True(t, errors.Is(err, ErrWorkflowAlreadyRunning))
		assert.Nil(t, lock2)

		// 刷新锁
		err = lock.Refresh(ctx, 10*time.Minute)
		assert.NoError(t, err)

		// 释放锁
		err = lock.Unlock(ctx)
		assert.NoError(t, err)

		// 现在第二个实例应该能获取锁
		lock3, err := workflowProvider2.LockWorkflowExecution(ctx, "test_execution_123")
		assert.NoError(t, err)
		assert.NotNil(t, lock3)

		// 清理
		err = lock3.Unlock(ctx)
		assert.NoError(t, err)
	})

	// 测试工作流模板锁
	t.Run("WorkflowTemplateLock", func(t *testing.T) {
		lock, err := workflowProvider.LockWorkflowTemplate(ctx, "test_template_456")
		assert.NoError(t, err)
		assert.NotNil(t, lock)

		// 验证锁信息
		assert.Equal(t, "workflow_template:test_template_456", lock.GetLockKey())
		assert.Equal(t, "test_template_456", lock.GetTemplateID())

		// 释放锁
		err = lock.Unlock(ctx)
		assert.NoError(t, err)
	})
}

func TestLockManager(t *testing.T) {
	db := setupTestDB(t)

	config := DefaultLockConfig()
	manager, err := NewLockManager(db, config)
	require.NoError(t, err)
	defer manager.Close()

	// 测试获取工作流锁提供者
	workflowProvider := manager.GetWorkflowLockProvider()
	assert.NotNil(t, workflowProvider)

	// 测试创建不同类型的锁提供者
	memoryProvider, err := manager.CreateLockProvider(LockProviderMemory)
	assert.NoError(t, err)
	assert.NotNil(t, memoryProvider)
	defer memoryProvider.Close()

	// 测试配置
	assert.Equal(t, LockProviderMySQL, manager.GetConfig().Provider)
}

func TestLockProviderFactory(t *testing.T) {
	db := setupTestDB(t)
	factory := NewLockProviderFactory(db)

	// 测试MySQL提供者
	t.Run("MySQL", func(t *testing.T) {
		config := &LockConfig{Provider: LockProviderMySQL}
		provider, err := factory.CreateLockProvider(config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		defer provider.Close()
	})

	// 测试内存提供者
	t.Run("Memory", func(t *testing.T) {
		config := &LockConfig{Provider: LockProviderMemory}
		provider, err := factory.CreateLockProvider(config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		defer provider.Close()
	})

	// 测试不支持的提供者
	t.Run("Unsupported", func(t *testing.T) {
		config := &LockConfig{Provider: "unsupported"}
		provider, err := factory.CreateLockProvider(config)
		assert.Error(t, err)
		assert.Nil(t, provider)
	})

	// 测试Redis提供者（尚未实现）
	t.Run("Redis", func(t *testing.T) {
		config := &LockConfig{Provider: LockProviderRedis}
		provider, err := factory.CreateLockProvider(config)
		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "Redis锁提供者尚未实现")
	})
}

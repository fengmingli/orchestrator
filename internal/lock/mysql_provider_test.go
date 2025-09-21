package lock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestMySQLProvider_BasicLockUnlock(t *testing.T) {
	db := setupTestDB(t)
	provider, err := NewMySQLLockProvider(db, nil)
	require.NoError(t, err)
	defer provider.Close()

	ctx := context.Background()

	opts := LockOptions{
		LockKey: "test_lock",
		Owner:   "instance_1",
		TTL:     30 * time.Second,
	}

	// 获取锁
	err = provider.Lock(ctx, opts)
	assert.NoError(t, err)

	// 检查锁是否存在
	locked, owner, err := provider.IsLocked(ctx, "test_lock")
	assert.NoError(t, err)
	assert.True(t, locked)
	assert.Equal(t, "instance_1", owner)

	// 释放锁
	err = provider.Unlock(ctx, "test_lock", "instance_1")
	assert.NoError(t, err)

	// 检查锁是否已释放
	locked, _, err = provider.IsLocked(ctx, "test_lock")
	assert.NoError(t, err)
	assert.False(t, locked)
}

func TestMySQLProvider_ConcurrentLock(t *testing.T) {
	db := setupTestDB(t)
	provider, err := NewMySQLLockProvider(db, nil)
	require.NoError(t, err)
	defer provider.Close()

	ctx := context.Background()

	opts1 := LockOptions{
		LockKey:    "test_concurrent",
		Owner:      "instance_1",
		TTL:        30 * time.Second,
		RetryCount: 0, // 不重试
	}

	opts2 := LockOptions{
		LockKey:    "test_concurrent",
		Owner:      "instance_2",
		TTL:        30 * time.Second,
		RetryCount: 0, // 不重试
	}

	// 第一个实例获取锁
	err = provider.Lock(ctx, opts1)
	assert.NoError(t, err)

	// 第二个实例尝试获取同一个锁，应该失败
	err = provider.Lock(ctx, opts2)
	assert.True(t, errors.Is(err, ErrLockTimeout))

	// 释放锁
	err = provider.Unlock(ctx, "test_concurrent", "instance_1")
	assert.NoError(t, err)

	// 现在第二个实例应该能获取锁
	err = provider.Lock(ctx, opts2)
	assert.NoError(t, err)

	// 清理
	err = provider.Unlock(ctx, "test_concurrent", "instance_2")
	assert.NoError(t, err)
}

func TestMySQLProvider_ReentrantLock(t *testing.T) {
	db := setupTestDB(t)
	provider, err := NewMySQLLockProvider(db, nil)
	require.NoError(t, err)
	defer provider.Close()

	ctx := context.Background()

	opts := LockOptions{
		LockKey: "test_reentrant",
		Owner:   "instance_1",
		TTL:     30 * time.Second,
	}

	// 获取锁
	err = provider.Lock(ctx, opts)
	assert.NoError(t, err)

	// 同一个owner再次获取锁（重入）
	err = provider.Lock(ctx, opts)
	assert.NoError(t, err)

	// 释放锁
	err = provider.Unlock(ctx, "test_reentrant", "instance_1")
	assert.NoError(t, err)
}

func TestMySQLProvider_ExpiredLock(t *testing.T) {
	db := setupTestDB(t)
	provider, err := NewMySQLLockProvider(db, nil)
	require.NoError(t, err)
	defer provider.Close()

	ctx := context.Background()

	opts := LockOptions{
		LockKey: "test_expired",
		Owner:   "instance_1",
		TTL:     100 * time.Millisecond, // 很短的TTL
	}

	// 获取锁
	err = provider.Lock(ctx, opts)
	assert.NoError(t, err)

	// 等待锁过期
	time.Sleep(200 * time.Millisecond)

	// 检查锁是否已过期
	locked, _, err := provider.IsLocked(ctx, "test_expired")
	assert.NoError(t, err)
	assert.False(t, locked)

	// 另一个实例应该能获取锁
	opts2 := LockOptions{
		LockKey: "test_expired",
		Owner:   "instance_2",
		TTL:     30 * time.Second,
	}
	err = provider.Lock(ctx, opts2)
	assert.NoError(t, err)

	// 清理
	err = provider.Unlock(ctx, "test_expired", "instance_2")
	assert.NoError(t, err)
}

func TestMySQLProvider_RefreshLock(t *testing.T) {
	db := setupTestDB(t)
	provider, err := NewMySQLLockProvider(db, nil)
	require.NoError(t, err)
	defer provider.Close()

	ctx := context.Background()

	opts := LockOptions{
		LockKey: "test_refresh",
		Owner:   "test_owner",
		TTL:     time.Second,
	}

	// 获取锁
	err = provider.Lock(ctx, opts)
	assert.NoError(t, err)

	// 刷新锁
	err = provider.RefreshLock(ctx, "test_refresh", "test_owner", 5*time.Minute)
	assert.NoError(t, err)

	// 清理
	err = provider.Unlock(ctx, "test_refresh", "test_owner")
	assert.NoError(t, err)
}

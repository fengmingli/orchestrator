package lock

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MemoryLock 内存锁结构
type MemoryLock struct {
	Owner     string
	ExpiresAt time.Time
}

// MemoryLockProvider 内存分布式锁提供者（仅用于测试）
type MemoryLockProvider struct {
	locks  map[string]*MemoryLock
	mutex  sync.RWMutex
	logger *logrus.Entry
}

// NewMemoryLockProvider 创建内存分布式锁提供者
func NewMemoryLockProvider() LockProvider {
	return &MemoryLockProvider{
		locks:  make(map[string]*MemoryLock),
		logger: logrus.NewEntry(logrus.New()).WithField("component", "memory_lock_provider"),
	}
}

// Lock 获取分布式锁
func (p *MemoryLockProvider) Lock(ctx context.Context, opts LockOptions) error {
	if opts.TTL <= 0 {
		opts.TTL = 30 * time.Second
	}
	if opts.RetryCount <= 0 {
		opts.RetryCount = 3
	}
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = time.Second
	}

	logger := p.logger.WithFields(logrus.Fields{
		"lock_key": opts.LockKey,
		"owner":    opts.Owner,
		"ttl":      opts.TTL,
	})

	for i := 0; i <= opts.RetryCount; i++ {
		if i > 0 {
			logger.WithField("retry_count", i).Info("重试获取分布式锁")
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(opts.RetryDelay):
			}
		}

		// 清理过期锁
		p.cleanExpiredLocks()

		// 尝试获取锁
		if p.tryLock(opts) {
			logger.Info("成功获取分布式锁")
			return nil
		}

		logger.WithField("retry_count", i).Info("锁已被其他实例持有，等待重试")
	}

	return ErrLockTimeout
}

// tryLock 尝试获取锁
func (p *MemoryLockProvider) tryLock(opts LockOptions) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	now := time.Now()
	expiresAt := now.Add(opts.TTL)

	// 检查锁是否存在
	if existingLock, exists := p.locks[opts.LockKey]; exists {
		// 如果锁已过期，更新锁
		if existingLock.ExpiresAt.Before(now) {
			p.locks[opts.LockKey] = &MemoryLock{
				Owner:     opts.Owner,
				ExpiresAt: expiresAt,
			}
			return true
		}

		// 如果是同一个owner，更新过期时间（重入锁）
		if existingLock.Owner == opts.Owner {
			existingLock.ExpiresAt = expiresAt
			return true
		}

		// 锁被其他owner持有
		return false
	}

	// 创建新锁
	p.locks[opts.LockKey] = &MemoryLock{
		Owner:     opts.Owner,
		ExpiresAt: expiresAt,
	}

	return true
}

// Unlock 释放分布式锁
func (p *MemoryLockProvider) Unlock(ctx context.Context, lockKey, owner string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger := p.logger.WithFields(logrus.Fields{
		"lock_key": lockKey,
		"owner":    owner,
	})

	lock, exists := p.locks[lockKey]
	if !exists {
		logger.Warn("尝试释放不存在的锁")
		return ErrLockNotFound
	}

	if lock.Owner != owner {
		logger.Warn("尝试释放不属于当前owner的锁")
		return ErrLockNotFound
	}

	delete(p.locks, lockKey)
	logger.Info("成功释放分布式锁")
	return nil
}

// RefreshLock 刷新锁的过期时间
func (p *MemoryLockProvider) RefreshLock(ctx context.Context, lockKey, owner string, ttl time.Duration) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if ttl <= 0 {
		ttl = 30 * time.Second
	}

	lock, exists := p.locks[lockKey]
	if !exists {
		return ErrLockNotFound
	}

	if lock.Owner != owner {
		return ErrLockNotFound
	}

	lock.ExpiresAt = time.Now().Add(ttl)
	return nil
}

// IsLocked 检查锁是否存在且未过期
func (p *MemoryLockProvider) IsLocked(ctx context.Context, lockKey string) (bool, string, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	lock, exists := p.locks[lockKey]
	if !exists {
		return false, "", nil
	}

	// 检查是否过期
	if lock.ExpiresAt.Before(time.Now()) {
		return false, "", nil
	}

	return true, lock.Owner, nil
}

// Close 关闭内存锁提供者
func (p *MemoryLockProvider) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.locks = make(map[string]*MemoryLock)
	p.logger.Info("内存锁提供者已关闭")
	return nil
}

// cleanExpiredLocks 清理过期的锁
func (p *MemoryLockProvider) cleanExpiredLocks() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	now := time.Now()
	cleanedCount := 0

	for key, lock := range p.locks {
		if lock.ExpiresAt.Before(now) {
			delete(p.locks, key)
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		p.logger.WithField("cleaned_count", cleanedCount).Info("清理过期锁")
	}
}
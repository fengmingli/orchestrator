package lock

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// DistributedLock 分布式锁模型
type DistributedLock struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	LockKey   string    `gorm:"uniqueIndex;size:255;not null" json:"lock_key"`
	Owner     string    `gorm:"size:255;not null" json:"owner"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (DistributedLock) TableName() string {
	return "distributed_locks"
}

// MySQLLockProvider MySQL分布式锁提供者
type MySQLLockProvider struct {
	db        *gorm.DB
	logger    *logrus.Entry
	tableName string
}

// NewMySQLLockProvider 创建MySQL分布式锁提供者
func NewMySQLLockProvider(db *gorm.DB, config *MySQLConfig) (LockProvider, error) {
	tableName := "distributed_locks"
	if config != nil && config.TableName != "" {
		tableName = config.TableName
	}

	provider := &MySQLLockProvider{
		db:        db,
		logger:    logrus.NewEntry(logrus.New()).WithField("component", "mysql_lock_provider"),
		tableName: tableName,
	}

	// 自动迁移锁表
	if err := db.Table(tableName).AutoMigrate(&DistributedLock{}); err != nil {
		provider.logger.WithError(err).Error("MySQL分布式锁表迁移失败")
		return nil, err
	}

	return provider, nil
}

// Lock 获取分布式锁
func (p *MySQLLockProvider) Lock(ctx context.Context, opts LockOptions) error {
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
		if err := p.cleanExpiredLocks(ctx); err != nil {
			logger.WithError(err).Warn("清理过期锁失败")
		}

		// 尝试获取锁
		if err := p.tryLock(ctx, opts); err == nil {
			logger.Info("成功获取分布式锁")
			return nil
		} else if err != ErrLockAlreadyExists {
			logger.WithError(err).Error("获取分布式锁失败")
			return err
		}

		logger.WithField("retry_count", i).Info("锁已被其他实例持有，等待重试")
	}

	return ErrLockTimeout
}

// tryLock 尝试获取锁
func (p *MySQLLockProvider) tryLock(ctx context.Context, opts LockOptions) error {
	now := time.Now()
	expiresAt := now.Add(opts.TTL)

	lock := &DistributedLock{
		LockKey:   opts.LockKey,
		Owner:     opts.Owner,
		ExpiresAt: expiresAt,
	}

	// 使用事务确保原子性
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 尝试插入新锁
		result := tx.Table(p.tableName).Create(lock)
		if result.Error != nil {
			// 检查是否是唯一约束冲突
			if isDuplicateKeyError(result.Error) {
				// 检查现有锁是否过期
				var existingLock DistributedLock
				if err := tx.Table(p.tableName).Where("lock_key = ?", opts.LockKey).First(&existingLock).Error; err != nil {
					return err
				}

				// 如果锁已过期，尝试更新
				if existingLock.ExpiresAt.Before(now) {
					updateResult := tx.Table(p.tableName).Model(&existingLock).
						Where("lock_key = ? AND expires_at < ?", opts.LockKey, now).
						Updates(map[string]interface{}{
							"owner":      opts.Owner,
							"expires_at": expiresAt,
							"updated_at": now,
						})
					if updateResult.Error != nil {
						return updateResult.Error
					}
					if updateResult.RowsAffected == 0 {
						return ErrLockAlreadyExists
					}
					return nil
				}

				// 如果是同一个owner，更新过期时间（重入锁）
				if existingLock.Owner == opts.Owner {
					return tx.Table(p.tableName).Model(&existingLock).Updates(map[string]interface{}{
						"expires_at": expiresAt,
						"updated_at": now,
					}).Error
				}

				return ErrLockAlreadyExists
			}
			return result.Error
		}

		return nil
	})
}

// Unlock 释放分布式锁
func (p *MySQLLockProvider) Unlock(ctx context.Context, lockKey, owner string) error {
	logger := p.logger.WithFields(logrus.Fields{
		"lock_key": lockKey,
		"owner":    owner,
	})

	// 只有锁的持有者才能释放锁
	result := p.db.WithContext(ctx).Table(p.tableName).
		Where("lock_key = ? AND owner = ?", lockKey, owner).
		Delete(&DistributedLock{})
	
	if result.Error != nil {
		logger.WithError(result.Error).Error("释放分布式锁失败")
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.Warn("尝试释放不存在或不属于当前owner的锁")
		return ErrLockNotFound
	}

	logger.Info("成功释放分布式锁")
	return nil
}

// RefreshLock 刷新锁的过期时间
func (p *MySQLLockProvider) RefreshLock(ctx context.Context, lockKey, owner string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}

	expiresAt := time.Now().Add(ttl)
	result := p.db.WithContext(ctx).Table(p.tableName).Model(&DistributedLock{}).
		Where("lock_key = ? AND owner = ?", lockKey, owner).
		Update("expires_at", expiresAt)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrLockNotFound
	}

	return nil
}

// IsLocked 检查锁是否存在且未过期
func (p *MySQLLockProvider) IsLocked(ctx context.Context, lockKey string) (bool, string, error) {
	var lock DistributedLock
	err := p.db.WithContext(ctx).Table(p.tableName).Where("lock_key = ?", lockKey).First(&lock).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, "", nil
		}
		return false, "", err
	}

	// 检查是否过期
	if lock.ExpiresAt.Before(time.Now()) {
		return false, "", nil
	}

	return true, lock.Owner, nil
}

// Close 关闭MySQL锁提供者
func (p *MySQLLockProvider) Close() error {
	// MySQL连接通常由外部管理，这里不需要关闭
	p.logger.Info("MySQL锁提供者已关闭")
	return nil
}

// cleanExpiredLocks 清理过期的锁
func (p *MySQLLockProvider) cleanExpiredLocks(ctx context.Context) error {
	result := p.db.WithContext(ctx).Table(p.tableName).
		Where("expires_at < ?", time.Now()).
		Delete(&DistributedLock{})
	
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		p.logger.WithField("cleaned_count", result.RowsAffected).Info("清理过期锁")
	}

	return nil
}

// 工具函数：检查是否是重复键错误
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	// MySQL duplicate entry error
	return containsAny(errStr, []string{
		"Duplicate entry",
		"duplicate key",
		"UNIQUE constraint failed",
	})
}

// 工具函数：检查字符串是否包含任意一个子字符串
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
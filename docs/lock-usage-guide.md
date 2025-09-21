# 分布式锁使用指南

## 快速开始

### 1. 基本使用（默认MySQL锁）

```go
package main

import (
    "github.com/fengmingli/orchestrator/internal/service"
    "gorm.io/gorm"
)

func main() {
    // 使用现有数据库连接
    var db *gorm.DB // 您的数据库连接
    
    // 创建编排服务（默认使用MySQL锁）
    orchestratorService := service.NewOrchestratorService(db)
    
    // 执行工作流（自动使用分布式锁）
    err := orchestratorService.ExecuteWorkflow("execution_123")
    if err != nil {
        // 处理错误
        log.Printf("执行工作流失败: %v", err)
    }
}
```

### 2. 自定义锁配置

```go
package main

import (
    "time"
    
    "github.com/fengmingli/orchestrator/internal/lock"
    "github.com/fengmingli/orchestrator/internal/service"
)

func main() {
    // 自定义锁配置
    lockConfig := &lock.LockConfig{
        Provider:          lock.LockProviderMySQL,
        DefaultTTL:        10 * time.Minute,  // 锁超时时间
        DefaultRetryCount: 5,                 // 重试次数
        DefaultRetryDelay: 2 * time.Second,   // 重试间隔
        MySQL: &lock.MySQLConfig{
            TableName: "my_custom_locks",     // 自定义表名
        },
    }
    
    // 使用自定义配置创建服务
    service := service.NewOrchestratorServiceWithLockConfig(db, lockConfig)
    
    // 执行工作流
    err := service.ExecuteWorkflow("execution_456")
    // ...
}
```

### 3. 测试环境使用内存锁

```go
func TestWorkflowExecution(t *testing.T) {
    // 测试环境使用内存锁，无需数据库
    testConfig := &lock.LockConfig{
        Provider: lock.LockProviderMemory,
    }
    
    service := service.NewOrchestratorServiceWithLockConfig(nil, testConfig)
    
    // 运行测试...
}
```

## 多副本部署场景

### 场景1：同一工作流在多个副本上触发

```bash
# 副本1
2025-09-21 14:30:00 INFO 开始执行工作流 execution_id=work_123
2025-09-21 14:30:00 INFO 成功获取分布式锁 lock_key=workflow_execution:work_123

# 副本2（几乎同时）
2025-09-21 14:30:01 INFO 开始执行工作流 execution_id=work_123
2025-09-21 14:30:01 INFO 工作流已被其他副本执行，跳过执行
```

**结果**：只有副本1执行工作流，副本2自动跳过，避免重复执行。

### 场景2：网络分区恢复后的锁处理

```bash
# 网络分区期间
2025-09-21 14:25:00 INFO 获取锁失败，网络异常

# 网络恢复后
2025-09-21 14:30:00 INFO 清理过期锁 cleaned_count=3
2025-09-21 14:30:00 INFO 成功获取分布式锁
```

**结果**：系统自动清理过期锁，确保服务恢复后正常工作。

## 错误处理

### 常见错误类型

```go
err := orchestratorService.ExecuteWorkflow("execution_123")
if err != nil {
    if errors.Is(err, lock.ErrWorkflowAlreadyRunning) {
        // 工作流已在其他实例运行，这是正常情况
        log.Info("工作流已在其他实例运行")
        return nil
    }
    
    if errors.Is(err, lock.ErrLockTimeout) {
        // 锁获取超时，可能需要增加重试次数
        log.Error("锁获取超时")
        return err
    }
    
    // 其他错误
    log.Error("执行工作流失败", err)
    return err
}
```

### 最佳实践

1. **错误比较使用 `errors.Is()`**：
   ```go
   // ✅ 正确
   if errors.Is(err, lock.ErrWorkflowAlreadyRunning) {
       // 处理逻辑
   }
   
   // ❌ 错误
   if err == lock.ErrWorkflowAlreadyRunning {
       // 可能不可靠
   }
   ```

2. **合理设置超时时间**：
   ```go
   lockConfig := &lock.LockConfig{
       DefaultTTL: 10 * time.Minute, // 根据工作流平均执行时间设置
   }
   ```

3. **监控锁状态**：
   ```go
   // 可以定期检查锁状态
   locked, owner, err := lockProvider.IsLocked(ctx, lockKey)
   if err == nil && locked {
       log.Printf("锁被 %s 持有", owner)
   }
   ```

## 性能调优

### MySQL锁性能优化

1. **数据库索引**：
   ```sql
   CREATE INDEX idx_lock_key ON distributed_locks(lock_key);
   CREATE INDEX idx_expires_at ON distributed_locks(expires_at);
   ```

2. **连接池配置**：
   ```go
   db.SetMaxOpenConns(25)
   db.SetMaxIdleConns(25)
   db.SetConnMaxLifetime(5 * time.Minute)
   ```

3. **锁超时设置**：
   ```go
   // 根据业务场景调整
   lockConfig.DefaultTTL = 5 * time.Minute  // 短任务
   lockConfig.DefaultTTL = 30 * time.Minute // 长任务
   ```

### 监控指标

```go
// 可以添加的监控指标
type LockMetrics struct {
    LockAcquisitionDuration time.Duration
    LockHoldDuration       time.Duration
    LockFailureCount       int64
    LockSuccessCount       int64
}
```

## 故障排查

### 常见问题

1. **锁获取总是失败**：
   - 检查数据库连接
   - 确认表是否正确创建
   - 查看锁是否被其他进程持有

2. **锁无法释放**：
   - 检查owner标识是否正确
   - 确认锁是否已过期

3. **死锁情况**：
   - 所有锁都有TTL自动过期
   - 可以手动清理：`DELETE FROM distributed_locks WHERE expires_at < NOW()`

### 调试技巧

```go
// 开启详细日志
logrus.SetLevel(logrus.DebugLevel)

// 查看锁状态
SELECT lock_key, owner, expires_at FROM distributed_locks WHERE lock_key LIKE 'workflow_%';

// 手动清理过期锁
DELETE FROM distributed_locks WHERE expires_at < NOW();
```

## 扩展新的锁实现

如需添加Redis锁支持：

1. **实现LockProvider接口**：
   ```go
   type RedisLockProvider struct {
       client *redis.Client
   }
   
   func (p *RedisLockProvider) Lock(ctx context.Context, opts LockOptions) error {
       // Redis SET NX EX 实现
   }
   ```

2. **注册到工厂**：
   ```go
   func (f *LockProviderFactory) createRedisProvider(config *RedisConfig) (LockProvider, error) {
       return NewRedisLockProvider(config), nil
   }
   ```

3. **添加配置支持**：
   ```go
   type LockConfig struct {
       Provider LockProviderType
       Redis    *RedisConfig // 新增Redis配置
   }
   ```

## 总结

通过使用抽象接口设计，您的系统现在具备了：

- ✅ **可插拔的锁实现**：轻松切换MySQL、Redis、etcd等
- ✅ **零配置启动**：默认使用MySQL锁，开箱即用
- ✅ **测试友好**：提供内存锁用于测试
- ✅ **生产就绪**：支持重入锁、自动过期、错误重试
- ✅ **监控友好**：详细的日志和状态检查

现在您可以放心在多副本环境中部署，系统将自动确保任务执行的唯一性。
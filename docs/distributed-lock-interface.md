# 分布式锁抽象接口设计

## 概述

为了支持不同的分布式锁实现（MySQL、Redis、etcd等），系统设计了一套抽象接口，实现了可插拔的分布式锁架构。上层业务代码依赖抽象接口，可以轻松切换不同的锁实现。

## 架构设计

### 核心接口

#### 1. LockProvider 接口
基础分布式锁提供者接口，定义了锁的基本操作：

```go
type LockProvider interface {
    Lock(ctx context.Context, opts LockOptions) error
    Unlock(ctx context.Context, lockKey, owner string) error
    RefreshLock(ctx context.Context, lockKey, owner string, ttl time.Duration) error
    IsLocked(ctx context.Context, lockKey string) (bool, string, error)
    Close() error
}
```

#### 2. WorkflowLockProvider 接口
工作流专用锁提供者接口：

```go
type WorkflowLockProvider interface {
    LockWorkflowExecution(ctx context.Context, executionID string) (WorkflowLockHandle, error)
    LockWorkflowTemplate(ctx context.Context, templateID string) (WorkflowLockHandle, error)
    Close() error
}
```

#### 3. WorkflowLockHandle 接口
工作流锁句柄接口：

```go
type WorkflowLockHandle interface {
    Unlock(ctx context.Context) error
    Refresh(ctx context.Context, ttl time.Duration) error
    GetLockKey() string
    GetExecutionID() string
    GetTemplateID() string
}
```

### 工厂模式

#### LockProviderFactory
负责根据配置创建不同类型的锁提供者：

```go
type LockProviderFactory struct {
    db *gorm.DB
}

func (f *LockProviderFactory) CreateLockProvider(config *LockConfig) (LockProvider, error)
func (f *LockProviderFactory) CreateWorkflowLockProvider(config *LockConfig) (WorkflowLockProvider, error)
```

#### LockManager
锁管理器，提供统一的锁服务入口：

```go
type LockManager struct {
    factory          *LockProviderFactory
    workflowProvider WorkflowLockProvider
    config           *LockConfig
}
```

## 支持的锁实现

### 1. MySQL 锁提供者 ✅
- **类型**: `LockProviderMySQL`
- **实现**: `MySQLLockProvider`
- **特性**: 基于数据库事务，支持重入锁，自动过期清理
- **适用场景**: 已有MySQL环境，无需额外组件

### 2. 内存锁提供者 ✅
- **类型**: `LockProviderMemory`  
- **实现**: `MemoryLockProvider`
- **特性**: 基于内存Map，快速响应
- **适用场景**: 测试环境，单机部署

### 3. Redis 锁提供者 🚧
- **类型**: `LockProviderRedis`
- **状态**: 待实现
- **特性**: 高性能，支持分布式
- **适用场景**: 高并发，低延迟场景

### 4. Etcd 锁提供者 🚧
- **类型**: `LockProviderEtcd`
- **状态**: 待实现  
- **特性**: 强一致性，支持监听
- **适用场景**: 需要强一致性的场景

## 配置管理

### 锁配置结构

```go
type LockConfig struct {
    Provider          LockProviderType  // 锁提供者类型
    MySQL            *MySQLConfig      // MySQL配置
    Redis            *RedisConfig      // Redis配置  
    Etcd             *EtcdConfig       // Etcd配置
    DefaultTTL       time.Duration     // 默认过期时间
    DefaultRetryCount int              // 默认重试次数
    DefaultRetryDelay time.Duration    // 默认重试延迟
}
```

### 默认配置

```go
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
```

## 使用方式

### 1. 使用默认配置

```go
// 使用默认MySQL锁提供者
service := NewOrchestratorService(db)
```

### 2. 使用自定义配置

```go
// 创建自定义锁配置
lockConfig := &lock.LockConfig{
    Provider: lock.LockProviderMySQL,
    DefaultTTL: 10 * time.Minute,
    MySQL: &lock.MySQLConfig{
        TableName: "custom_locks",
    },
}

// 使用自定义配置创建服务
service := NewOrchestratorServiceWithLockConfig(db, lockConfig)
```

### 3. 直接使用锁管理器

```go
// 创建锁管理器
lockManager, err := lock.NewLockManager(db, lockConfig)
if err != nil {
    return err
}
defer lockManager.Close()

// 获取工作流锁提供者
workflowProvider := lockManager.GetWorkflowLockProvider()

// 锁定工作流执行
lockHandle, err := workflowProvider.LockWorkflowExecution(ctx, "execution_123")
if err != nil {
    return err
}
defer lockHandle.Unlock(ctx)

// 执行业务逻辑...
```

### 4. 切换锁实现

```go
// 从MySQL切换到内存锁（测试环境）
testConfig := &lock.LockConfig{
    Provider: lock.LockProviderMemory,
}

testService := NewOrchestratorServiceWithLockConfig(nil, testConfig)
```

## 扩展新的锁实现

### 步骤1: 实现LockProvider接口

```go
type RedisLockProvider struct {
    client *redis.Client
    logger *logrus.Entry
}

func (p *RedisLockProvider) Lock(ctx context.Context, opts LockOptions) error {
    // 实现Redis锁逻辑
}

func (p *RedisLockProvider) Unlock(ctx context.Context, lockKey, owner string) error {
    // 实现Redis解锁逻辑  
}

// ... 实现其他方法
```

### 步骤2: 注册到工厂

```go
func (f *LockProviderFactory) createRedisProvider(config *RedisConfig) (LockProvider, error) {
    client := redis.NewClient(&redis.Options{
        Addr:     config.Addr,
        Password: config.Password,
        DB:       config.DB,
    })
    
    return &RedisLockProvider{
        client: client,
        logger: logrus.NewEntry(logrus.New()),
    }, nil
}
```

### 步骤3: 更新工厂方法

```go
func (f *LockProviderFactory) CreateLockProvider(config *LockConfig) (LockProvider, error) {
    switch config.Provider {
    case LockProviderRedis:
        return f.createRedisProvider(config.Redis)
    // ... 其他实现
    }
}
```

## 最佳实践

### 1. 锁粒度选择
- **执行锁**: 按executionID加锁，确保同一工作流不重复执行
- **模板锁**: 按templateID加锁，控制模板级别的并发
- **资源锁**: 按资源ID加锁，保护共享资源

### 2. 超时时间设置  
- **短期任务**: 1-5分钟
- **长期任务**: 10-30分钟
- **定期刷新**: 对于长时间运行的任务，定期刷新锁

### 3. 错误处理
- 区分锁超时和系统错误
- 合理设置重试策略
- 记录详细的锁操作日志

### 4. 测试策略
- 使用内存锁进行单元测试
- 使用MySQL锁进行集成测试
- 模拟网络分区和故障场景

## 性能考虑

### MySQL 锁
- **延迟**: 10-50ms
- **吞吐**: 1000-5000 ops/s
- **优势**: 无需额外组件，事务一致性
- **劣势**: 相对较慢

### Redis 锁（待实现）
- **延迟**: 1-5ms  
- **吞吐**: 10000+ ops/s
- **优势**: 高性能，低延迟
- **劣势**: 需要Redis集群

### 内存锁
- **延迟**: <1ms
- **吞吐**: 100000+ ops/s  
- **优势**: 极高性能
- **劣势**: 仅限单机，重启丢失

## 监控和运维

### 关键指标
- 锁获取成功率
- 锁持有时间分布
- 锁竞争情况
- 过期锁清理频率

### 告警设置
- 锁获取失败率超过阈值
- 锁持有时间过长
- 死锁检测

### 故障排查
- 检查锁表状态
- 分析锁竞争日志
- 监控锁提供者健康状态
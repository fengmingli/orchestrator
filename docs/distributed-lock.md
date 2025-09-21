# 分布式锁使用指南

## 概述

本系统实现了基于MySQL的分布式锁机制，确保在多副本部署环境下，同一个工作流执行只会在一个副本上运行，避免重复执行。

## 特性

- **基于MySQL的分布式锁**：利用现有数据库，无需额外组件
- **自动过期机制**：锁支持TTL，防止死锁
- **重入锁支持**：同一owner可以重复获取锁
- **锁刷新功能**：支持延长锁的过期时间
- **并发安全**：使用数据库事务保证原子性

## 核心组件

### 1. DistributedLockManager
基础分布式锁管理器，提供锁的基本操作。

### 2. WorkflowLockManager  
工作流专用锁管理器，针对工作流执行场景优化。

### 3. WorkflowLock
工作流锁对象，提供锁的生命周期管理。

## 使用方式

### 在OrchestratorService中的集成

系统已经自动集成了分布式锁，在执行工作流时会自动获取锁：

```go
// ExecuteWorkflow 执行工作流
func (s *OrchestratorService) ExecuteWorkflow(executionID string) error {
    ctx := context.Background()
    
    // 获取分布式锁，确保只有一个副本执行
    workflowLock, err := s.lockManager.LockWorkflowExecution(ctx, executionID)
    if err != nil {
        if err == lock.ErrWorkflowAlreadyRunning {
            // 工作流已被其他副本执行，跳过
            return nil
        }
        return err
    }

    // 确保在函数结束时释放锁
    defer workflowLock.Unlock(ctx)

    // 执行工作流...
}
```

### 手动使用分布式锁

如果需要在其他场景使用分布式锁：

```go
// 创建锁管理器
lockManager := lock.NewDistributedLockManager(db)

// 定义锁选项
opts := lock.LockOptions{
    LockKey:    "my_task_123",
    Owner:      "instance_1", 
    TTL:        5 * time.Minute,
    RetryCount: 3,
    RetryDelay: time.Second,
}

// 获取锁
if err := lockManager.Lock(ctx, opts); err != nil {
    if err == lock.ErrLockTimeout {
        // 锁获取超时
        return
    }
    // 其他错误
    return
}

// 执行业务逻辑...

// 释放锁
lockManager.Unlock(ctx, "my_task_123", "instance_1")
```

## 锁的类型

### 1. 执行锁 (Execution Lock)
- **用途**：确保同一个工作流执行只在一个副本上运行
- **锁键格式**：`workflow_execution:{executionID}`
- **TTL**：5分钟
- **重试**：不重试（避免重复执行）

### 2. 模板锁 (Template Lock)  
- **用途**：确保同一个模板的多个实例不会同时执行
- **锁键格式**：`workflow_template:{templateID}`
- **TTL**：10分钟
- **重试**：重试2次

## 数据库表结构

```sql
CREATE TABLE distributed_locks (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    lock_key   VARCHAR(255) NOT NULL UNIQUE,
    owner      VARCHAR(255) NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
```

## 部署配置

### 多副本部署
在多副本环境下，每个副本会自动生成唯一的实例ID：
- 格式：`{hostname}-{uuid前8位}`
- 例如：`web-server-1-a1b2c3d4`

### 环境变量
无需额外配置，系统会自动使用现有的MySQL连接。

## 监控和日志

系统会输出详细的锁操作日志：

```
level=info msg="成功获取分布式锁" component=distributed_lock lock_key=workflow_execution:exec_123 owner=web-1-a1b2c3d4
level=info msg="工作流已被其他副本执行，跳过" component=workflow_lock execution_id=exec_123
level=info msg="成功释放分布式锁" component=distributed_lock lock_key=workflow_execution:exec_123
```

## 故障处理

### 锁泄漏
- 锁有自动过期机制，最长5-10分钟自动释放
- 系统会自动清理过期锁

### 死锁检测
- 通过TTL机制避免死锁
- 支持锁刷新延长执行时间

### 网络分区
- 基于数据库的锁机制，天然支持网络分区恢复

## 性能考虑

- 锁操作使用数据库索引，性能良好
- 支持高并发场景
- 自动清理机制避免锁表膨胀

## 测试

运行测试验证分布式锁功能：

```bash
go test ./internal/lock -v
```

测试覆盖：
- 基本锁获取和释放
- 并发锁竞争
- 重入锁机制  
- 锁过期处理
- 工作流锁管理
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/fengmingli/orchestrator/internal/lock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 初始化数据库（示例使用SQLite）
	db, err := gorm.Open(sqlite.Open("example.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 示例1: 使用MySQL锁提供者
	fmt.Println("=== 示例1: MySQL锁提供者 ===")
	mysqlExample(db)

	// 示例2: 使用内存锁提供者
	fmt.Println("\n=== 示例2: 内存锁提供者 ===")
	memoryExample()

	// 示例3: 工作流锁使用
	fmt.Println("\n=== 示例3: 工作流锁 ===")
	workflowExample(db)

	// 示例4: 锁管理器使用
	fmt.Println("\n=== 示例4: 锁管理器 ===")
	lockManagerExample(db)

	// 示例5: 并发锁竞争
	fmt.Println("\n=== 示例5: 并发锁竞争 ===")
	concurrentExample(db)
}

// 示例1: MySQL锁提供者基本使用
func mysqlExample(db *gorm.DB) {
	ctx := context.Background()

	// 创建MySQL锁提供者
	provider, err := lock.NewMySQLLockProvider(db, &lock.MySQLConfig{
		TableName: "example_locks",
	})
	if err != nil {
		log.Fatal("创建MySQL锁提供者失败:", err)
	}
	defer provider.Close()

	// 定义锁选项
	opts := lock.LockOptions{
		LockKey:    "mysql_example_lock",
		Owner:      "instance_1",
		TTL:        30 * time.Second,
		RetryCount: 3,
		RetryDelay: time.Second,
	}

	// 获取锁
	fmt.Printf("尝试获取锁: %s\n", opts.LockKey)
	if err := provider.Lock(ctx, opts); err != nil {
		log.Printf("获取锁失败: %v\n", err)
		return
	}
	fmt.Printf("成功获取锁: %s\n", opts.LockKey)

	// 检查锁状态
	locked, owner, err := provider.IsLocked(ctx, opts.LockKey)
	if err != nil {
		log.Printf("检查锁状态失败: %v\n", err)
	} else {
		fmt.Printf("锁状态: locked=%t, owner=%s\n", locked, owner)
	}

	// 模拟业务处理
	fmt.Println("执行业务逻辑...")
	time.Sleep(2 * time.Second)

	// 刷新锁
	fmt.Println("刷新锁过期时间...")
	if err := provider.RefreshLock(ctx, opts.LockKey, opts.Owner, 60*time.Second); err != nil {
		log.Printf("刷新锁失败: %v\n", err)
	}

	// 释放锁
	fmt.Printf("释放锁: %s\n", opts.LockKey)
	if err := provider.Unlock(ctx, opts.LockKey, opts.Owner); err != nil {
		log.Printf("释放锁失败: %v\n", err)
	}
}

// 示例2: 内存锁提供者使用
func memoryExample() {
	ctx := context.Background()

	// 创建内存锁提供者
	provider := lock.NewMemoryLockProvider()
	defer provider.Close()

	opts := lock.LockOptions{
		LockKey: "memory_example_lock",
		Owner:   "instance_1",
		TTL:     10 * time.Second,
	}

	fmt.Printf("使用内存锁获取锁: %s\n", opts.LockKey)
	if err := provider.Lock(ctx, opts); err != nil {
		log.Printf("获取锁失败: %v\n", err)
		return
	}

	fmt.Println("成功获取内存锁，执行业务逻辑...")
	time.Sleep(1 * time.Second)

	if err := provider.Unlock(ctx, opts.LockKey, opts.Owner); err != nil {
		log.Printf("释放锁失败: %v\n", err)
	}
	fmt.Println("成功释放内存锁")
}

// 示例3: 工作流锁使用
func workflowExample(db *gorm.DB) {
	ctx := context.Background()

	// 创建锁配置
	config := &lock.LockConfig{
		Provider:   lock.LockProviderMySQL,
		DefaultTTL: 5 * time.Minute,
		MySQL: &lock.MySQLConfig{
			TableName: "workflow_locks",
		},
	}

	// 创建锁工厂
	factory := lock.NewLockProviderFactory(db)
	workflowProvider, err := factory.CreateWorkflowLockProvider(config)
	if err != nil {
		log.Fatal("创建工作流锁提供者失败:", err)
	}
	defer workflowProvider.Close()

	// 锁定工作流执行
	fmt.Println("锁定工作流执行...")
	executionLock, err := workflowProvider.LockWorkflowExecution(ctx, "example_execution_123")
	if err != nil {
		if errors.Is(err, lock.ErrWorkflowAlreadyRunning) {
			fmt.Println("工作流已在其他实例运行")
			return
		}
		log.Printf("锁定工作流执行失败: %v\n", err)
		return
	}

	fmt.Printf("成功锁定工作流执行, 锁键: %s\n", executionLock.GetLockKey())
	fmt.Printf("执行ID: %s\n", executionLock.GetExecutionID())

	// 模拟工作流执行
	fmt.Println("执行工作流...")
	time.Sleep(2 * time.Second)

	// 释放锁
	if err := executionLock.Unlock(ctx); err != nil {
		log.Printf("释放工作流锁失败: %v\n", err)
	}
	fmt.Println("工作流执行完成，锁已释放")
}

// 示例4: 锁管理器使用
func lockManagerExample(db *gorm.DB) {
	ctx := context.Background()

	// 创建锁管理器
	lockManager, err := lock.NewLockManager(db, nil) // 使用默认配置
	if err != nil {
		log.Fatal("创建锁管理器失败:", err)
	}
	defer lockManager.Close()

	// 获取工作流锁提供者
	workflowProvider := lockManager.GetWorkflowLockProvider()

	// 尝试锁定模板
	fmt.Println("锁定工作流模板...")
	templateLock, err := workflowProvider.LockWorkflowTemplate(ctx, "example_template_456")
	if err != nil {
		log.Printf("锁定工作流模板失败: %v\n", err)
		return
	}

	fmt.Printf("成功锁定工作流模板, 模板ID: %s\n", templateLock.GetTemplateID())

	// 执行模板相关操作
	fmt.Println("执行模板操作...")
	time.Sleep(1 * time.Second)

	// 释放模板锁
	if err := templateLock.Unlock(ctx); err != nil {
		log.Printf("释放模板锁失败: %v\n", err)
	}
	fmt.Println("模板操作完成，锁已释放")
}

// 示例5: 并发锁竞争
func concurrentExample(db *gorm.DB) {
	ctx := context.Background()

	// 创建两个锁提供者模拟不同实例
	provider1, _ := lock.NewMySQLLockProvider(db, &lock.MySQLConfig{
		TableName: "concurrent_locks",
	})
	defer provider1.Close()

	provider2, _ := lock.NewMySQLLockProvider(db, &lock.MySQLConfig{
		TableName: "concurrent_locks",
	})
	defer provider2.Close()

	lockKey := "concurrent_example_lock"

	// 启动第一个goroutine
	go func() {
		opts := lock.LockOptions{
			LockKey:    lockKey,
			Owner:      "instance_1",
			TTL:        5 * time.Second,
			RetryCount: 2,
		}

		fmt.Println("[Instance 1] 尝试获取锁...")
		if err := provider1.Lock(ctx, opts); err != nil {
			fmt.Printf("[Instance 1] 获取锁失败: %v\n", err)
			return
		}
		fmt.Println("[Instance 1] 成功获取锁")

		// 持有锁3秒
		time.Sleep(3 * time.Second)

		fmt.Println("[Instance 1] 释放锁")
		provider1.Unlock(ctx, lockKey, "instance_1")
	}()

	// 等待一秒，然后启动第二个goroutine
	time.Sleep(1 * time.Second)

	go func() {
		opts := lock.LockOptions{
			LockKey:    lockKey,
			Owner:      "instance_2",
			TTL:        5 * time.Second,
			RetryCount: 5,
			RetryDelay: time.Second,
		}

		fmt.Println("[Instance 2] 尝试获取锁...")
		if err := provider2.Lock(ctx, opts); err != nil {
			fmt.Printf("[Instance 2] 获取锁失败: %v\n", err)
			return
		}
		fmt.Println("[Instance 2] 成功获取锁")

		time.Sleep(1 * time.Second)

		fmt.Println("[Instance 2] 释放锁")
		provider2.Unlock(ctx, lockKey, "instance_2")
	}()

	// 等待所有goroutine完成
	time.Sleep(8 * time.Second)
	fmt.Println("并发示例完成")
}
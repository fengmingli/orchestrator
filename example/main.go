package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fengmingli/orchestrator/internal/engine"
	"github.com/fengmingli/orchestrator/internal/engine/executorx"
	"github.com/fengmingli/orchestrator/internal/engine/task"
	"github.com/fengmingli/orchestrator/internal/engine/workflow"
	"github.com/sirupsen/logrus"
)

func main() {
	fmt.Println("=== 任务编排器演示 ===")

	// 1. 基本功能演示
	demonstrateBasicOrchestration()

	// 2. 复杂工作流演示
	demonstrateComplexWorkflow()

	// 3. HTTP任务演示
	demonstrateHTTPTask()

	// 4. Shell任务演示
	demonstrateShellTask()

	// 5. 错误处理演示
	demonstrateErrorHandling()
}

// demonstrateBasicOrchestration 演示基本编排功能
func demonstrateBasicOrchestration() {
	fmt.Println("\n--- 1. 基本任务编排演示 ---")

	// 创建编排器
	orchestrator := engine.NewTaskOrchestrator().
		WithMaxWorkers(5).
		WithRetryConfig(3, time.Second)

	// 创建简单的函数任务
	task1 := task.NewFuncTask("task1", "数据准备", func(ctx context.Context) (interface{}, error) {
		fmt.Println("正在准备数据...")
		time.Sleep(100 * time.Millisecond)
		return "数据准备完成", nil
	})

	task2 := task.NewFuncTask("task2", "数据处理", func(ctx context.Context) (interface{}, error) {
		fmt.Println("正在处理数据...")
		time.Sleep(200 * time.Millisecond)
		return "数据处理完成", nil
	})

	task3 := task.NewFuncTask("task3", "结果保存", func(ctx context.Context) (interface{}, error) {
		fmt.Println("正在保存结果...")
		time.Sleep(50 * time.Millisecond)
		return "结果保存完成", nil
	})

	// 创建串行工作流
	definitions, err := orchestrator.CreateSimpleWorkflow(task1, task2, task3)
	if err != nil {
		log.Fatal(err)
	}

	// 执行工作流
	ctx := context.Background()
	start := time.Now()
	result, err := orchestrator.Execute(ctx, definitions)
	duration := time.Since(start)

	if err != nil {
		log.Printf("执行失败: %v", err)
	} else {
		fmt.Printf("执行成功! 总耗时: %v\n", duration)
		fmt.Printf("任务结果数量: %d\n", len(result.TaskResults))
	}
}

// demonstrateComplexWorkflow 演示复杂工作流
func demonstrateComplexWorkflow() {
	fmt.Println("\n--- 2. 复杂依赖工作流演示 ---")

	orchestrator := engine.NewTaskOrchestrator()

	// 创建具有复杂依赖关系的任务 A -> [B, C] -> D
	taskA := task.NewFuncTask("A", "初始化", func(ctx context.Context) (interface{}, error) {
		fmt.Println("任务A: 系统初始化...")
		time.Sleep(50 * time.Millisecond)
		return "初始化完成", nil
	})

	taskB := task.NewFuncTask("B", "用户服务", func(ctx context.Context) (interface{}, error) {
		fmt.Println("任务B: 启动用户服务...")
		time.Sleep(80 * time.Millisecond)
		return "用户服务启动", nil
	})

	taskC := task.NewFuncTask("C", "订单服务", func(ctx context.Context) (interface{}, error) {
		fmt.Println("任务C: 启动订单服务...")
		time.Sleep(60 * time.Millisecond)
		return "订单服务启动", nil
	})

	taskD := task.NewFuncTask("D", "健康检查", func(ctx context.Context) (interface{}, error) {
		fmt.Println("任务D: 系统健康检查...")
		time.Sleep(30 * time.Millisecond)
		return "系统正常", nil
	})

	// 手动定义复杂依赖关系
	definitions := []engine.TaskDefinition{
		{
			ID:           "A",
			Name:         "系统初始化",
			Task:         taskA,
			Dependencies: nil, // 无依赖
			Mode:         workflow.RunModeSerial,
			Policy:       workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort},
		},
		{
			ID:           "B",
			Name:         "用户服务",
			Task:         taskB,
			Dependencies: []string{"A"},            // 依赖A
			Mode:         workflow.RunModeParallel, // B和C并行
			Policy:       workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort},
		},
		{
			ID:           "C",
			Name:         "订单服务",
			Task:         taskC,
			Dependencies: []string{"A"},            // 依赖A
			Mode:         workflow.RunModeParallel, // B和C并行
			Policy:       workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort},
		},
		{
			ID:           "D",
			Name:         "健康检查",
			Task:         taskD,
			Dependencies: []string{"B", "C"}, // 依赖B和C
			Mode:         workflow.RunModeSerial,
			Policy:       workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort},
		},
	}

	// 执行复杂工作流
	ctx := context.Background()
	start := time.Now()
	result, err := orchestrator.Execute(ctx, definitions)
	duration := time.Since(start)

	if err != nil {
		log.Printf("复杂工作流执行失败: %v", err)
	} else {
		fmt.Printf("复杂工作流执行成功! 总耗时: %v\n", duration)
		fmt.Printf("任务数量: %d, 预期时间约: A(50ms) + max(B(80ms), C(60ms)) + D(30ms) ≈ 160ms\n", len(result.TaskResults))
	}
}

// demonstrateHTTPTask 演示HTTP任务
func demonstrateHTTPTask() {
	fmt.Println("\n--- 3. HTTP任务演示 ---")

	orchestrator := engine.NewTaskOrchestrator()

	// 创建HTTP任务
	httpTask := task.NewHTTPTask("api-check", "API健康检查", "GET", "https://httpbin.org/status/200")

	definitions := []engine.TaskDefinition{
		{
			ID:     "api-check",
			Name:   "API健康检查",
			Task:   httpTask,
			Mode:   workflow.RunModeSerial,
			Policy: workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort},
		},
	}

	ctx := context.Background()
	result, err := orchestrator.Execute(ctx, definitions)

	if err != nil {
		log.Printf("HTTP任务执行失败: %v", err)
	} else {
		fmt.Printf("HTTP任务执行成功! 状态: %v\n", result.Success)
	}
}

// demonstrateShellTask 演示Shell任务
func demonstrateShellTask() {
	fmt.Println("\n--- 4. Shell任务演示 ---")

	orchestrator := engine.NewTaskOrchestrator()

	// 创建Shell任务
	shellTask := task.NewShellTask("system-info", "系统信息", "echo '系统信息:'; uname -a; echo '当前时间:'; date")

	definitions := []engine.TaskDefinition{
		{
			ID:     "system-info",
			Name:   "获取系统信息",
			Task:   shellTask,
			Mode:   workflow.RunModeSerial,
			Policy: workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort},
		},
	}

	ctx := context.Background()
	result, err := orchestrator.Execute(ctx, definitions)

	if err != nil {
		log.Printf("Shell任务执行失败: %v", err)
	} else {
		fmt.Printf("Shell任务执行成功! 状态: %v\n", result.Success)
	}
}

// demonstrateErrorHandling 演示错误处理
func demonstrateErrorHandling() {
	fmt.Println("\n--- 5. 错误处理演示 ---")

	// 创建带日志的编排器
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // 只显示警告和错误
	logEntry := logrus.NewEntry(logger)

	orchestrator := engine.NewTaskOrchestrator().
		WithLogger(logEntry).
		AddHook(executorx.NewLoggingHook(logEntry))

	// 创建会失败的任务和正常任务
	failingTask := task.NewFuncTask("failing", "失败任务", func(ctx context.Context) (interface{}, error) {
		return nil, fmt.Errorf("模拟任务失败")
	})

	successTask := task.NewFuncTask("success", "成功任务", func(ctx context.Context) (interface{}, error) {
		fmt.Println("成功任务正常执行")
		return "任务完成", nil
	})

	// 测试失败中止策略
	fmt.Println("\n5.1 失败中止策略:")
	definitions := []engine.TaskDefinition{
		{
			ID:     "failing",
			Task:   failingTask,
			Policy: workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort},
		},
		{
			ID:           "success",
			Task:         successTask,
			Dependencies: []string{"failing"},
			Policy:       workflow.ExecutionPolicy{OnFailure: workflow.FailureAbort},
		},
	}

	ctx := context.Background()
	result, err := orchestrator.Execute(ctx, definitions)
	if err != nil {
		fmt.Printf("预期失败: %v\n", err)
	}

	// 测试失败跳过策略
	fmt.Println("\n5.2 失败跳过策略:")
	definitions[0].Policy = workflow.ExecutionPolicy{OnFailure: workflow.FailureSkip}

	result, err = orchestrator.Execute(ctx, definitions)
	if err != nil {
		fmt.Printf("意外失败: %v\n", err)
	} else {
		fmt.Printf("跳过失败策略执行成功! 状态: %v\n", result.Success)
	}
}

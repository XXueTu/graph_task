package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/XXueTu/graph_task/domain/workflow"
	"github.com/XXueTu/graph_task/engine"
)

func main() {
	fmt.Println("🚀 启动全新分层架构任务编排引擎")

	// 创建引擎配置
	config := engine.DefaultEngineConfig()
	config.MySQLDSN = "root:Root@123@tcp(10.99.51.9:3306)/graph_task?charset=utf8mb4&parseTime=True&loc=Local"

	// 创建引擎
	engine, err := engine.NewEngine(config)
	if err != nil {
		log.Fatalf("❌ 创建引擎失败: %v", err)
	}
	defer engine.Close()

	fmt.Println("✅ 引擎创建成功")

	// 演示基本功能
	if err := demonstrateBasicFeatures(engine); err != nil {
		log.Fatalf("❌ 演示失败: %v", err)
	}

	fmt.Println("🎉 新架构演示完成！")
}

func demonstrateBasicFeatures(engine engine.Engine) error {
	ctx := context.Background()

	// 1. 创建和发布工作流
	fmt.Println("\n📋 步骤1: 创建和发布工作流")

	builder := engine.CreateWorkflow("demo-workflow")
	workflow, err := builder.
		SetDescription("演示工作流 - 展示新架构功能").
		SetVersion("1.0.0").
		AddTask("task1", "数据预处理", preprocessTask).
		AddTask("task2", "数据处理", processTask).
		AddTask("task3", "数据后处理", postprocessTask).
		AddDependency("task1", "task2").
		AddDependency("task2", "task3").
		Build()

	if err != nil {
		return fmt.Errorf("构建工作流失败: %w", err)
	}

	if err := engine.PublishWorkflow(workflow); err != nil {
		return fmt.Errorf("发布工作流失败: %w", err)
	}

	fmt.Printf("✅ 工作流已创建并发布: %s\n", workflow.ID())

	// 2. 执行工作流
	fmt.Println("\n🔄 步骤2: 执行工作流")

	input := map[string]interface{}{
		"data":   []interface{}{1, 2, 3, 4, 5},
		"factor": 2,
	}

	execution, err := engine.Execute(ctx, workflow.ID(), input)
	if err != nil {
		return fmt.Errorf("执行工作流失败: %w", err)
	}

	fmt.Printf("✅ 工作流执行完成: %s\n", execution.ID())
	fmt.Printf("   状态: %s\n", execution.Status())
	fmt.Printf("   耗时: %v\n", execution.Duration())

	// 3. 监控和日志
	fmt.Println("\n📊 步骤3: 检查执行状态和日志")

	// 获取执行状态
	status, err := engine.GetExecutionStatus(execution.ID())
	if err != nil {
		fmt.Printf("⚠️  获取执行状态失败: %v\n", err)
	} else {
		fmt.Printf("✅ 执行进度: %.1f%%\n", status.Progress())
		fmt.Printf("   完成任务: %d/%d\n", status.CompletedTasks(), status.TotalTasks())
	}

	// 获取执行日志
	logs, err := engine.GetExecutionLogs(execution.ID(), 10, 0)
	if err != nil {
		fmt.Printf("⚠️  获取执行日志失败: %v\n", err)
	} else {
		fmt.Printf("✅ 执行日志条数: %d\n", len(logs))
	}

	// 4. 列出工作流
	fmt.Println("\n📜 步骤4: 列出所有工作流")

	workflows, err := engine.ListWorkflows()
	if err != nil {
		return fmt.Errorf("列出工作流失败: %w", err)
	}

	fmt.Printf("✅ 共有 %d 个工作流:\n", len(workflows))
	for i, wf := range workflows {
		fmt.Printf("   %d. %s (%s) - %s\n", i+1, wf.Name(), wf.ID(), wf.Status())
	}

	// 5. 演示异步执行
	fmt.Println("\n⚡ 步骤5: 演示异步执行")

	asyncInput := map[string]interface{}{
		"data":   []interface{}{10, 20, 30},
		"factor": 3,
	}

	executionID, err := engine.ExecuteAsync(ctx, workflow.ID(), asyncInput)
	if err != nil {
		return fmt.Errorf("异步执行失败: %w", err)
	}

	fmt.Printf("✅ 异步执行已启动: %s\n", executionID)

	// 等待异步执行完成
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		asyncExecution, err := engine.GetExecution(executionID)
		if err != nil {
			continue
		}

		fmt.Printf("   异步执行状态: %s\n", asyncExecution.Status())
		if asyncExecution.IsCompleted() {
			break
		}
	}

	fmt.Println("\n🎯 新架构演示完成 - 所有核心功能正常工作!")
	return nil
}

// 示例任务处理器
func preprocessTask(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	fmt.Println("  🔧 执行数据预处理...")
	time.Sleep(200 * time.Millisecond) // 模拟处理时间

	data, ok := input["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("无效的数据类型")
	}

	// 转换为整数数组
	var intData []int
	for _, v := range data {
		if intVal, ok := v.(int); ok {
			intData = append(intData, intVal)
		} else if floatVal, ok := v.(float64); ok {
			intData = append(intData, int(floatVal))
		}
	}

	// 简单的预处理：过滤负数
	var processed []int
	for _, v := range intData {
		if v >= 0 {
			processed = append(processed, v)
		}
	}

	return map[string]interface{}{
		"processed_data": processed,
		"original_count": len(intData),
		"filtered_count": len(processed),
	}, nil
}

func processTask(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	fmt.Println("  ⚙️  执行数据处理...")
	time.Sleep(300 * time.Millisecond) // 模拟处理时间

	data, ok := input["task1_output"].(map[string]interface{})
	if !ok {
		// 如果没有依赖任务的输出，使用直接输入
		data = input
	}

	processedData, ok := data["processed_data"].([]int)
	if !ok {
		return nil, fmt.Errorf("无法获取预处理数据")
	}

	factor, ok := input["factor"].(int)
	if !ok {
		factor = 1
	}

	// 处理数据：乘以因子
	var result []int
	sum := 0
	for _, v := range processedData {
		processed := v * factor
		result = append(result, processed)
		sum += processed
	}

	return map[string]interface{}{
		"result":  result,
		"sum":     sum,
		"average": float64(sum) / float64(len(result)),
	}, nil
}

func postprocessTask(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	fmt.Println("  📋 执行数据后处理...")
	time.Sleep(150 * time.Millisecond) // 模拟处理时间

	data, ok := input["task2_output"].(map[string]interface{})
	if !ok {
		data = input
	}

	result, ok := data["result"].([]int)
	if !ok {
		return nil, fmt.Errorf("无法获取处理结果")
	}

	// 后处理：生成统计信息
	min, max := result[0], result[0]
	for _, v := range result {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	return map[string]interface{}{
		"final_result": result,
		"statistics": map[string]interface{}{
			"min":   min,
			"max":   max,
			"count": len(result),
		},
		"status": "completed",
	}, nil
}

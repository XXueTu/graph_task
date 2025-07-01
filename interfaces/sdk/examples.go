package sdk

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/XXueTu/graph_task/domain/workflow"
)

// QuickStartExample 快速开始示例
func QuickStartExample() error {
	// 1. 创建客户端
	client, err := NewSimpleClient()
	if err != nil {
		return fmt.Errorf("创建客户端失败: %w", err)
	}
	defer client.Close()

	// 2. 创建工作流
	builder := client.CreateWorkflow("quick-start")
	workflow, err := builder.
		SetDescription("快速开始示例工作流").
		SetVersion("1.0.0").
		AddTask("hello", "打招呼", helloTaskHandler).
		AddTask("process", "处理数据", processTaskHandler).
		AddTask("goodbye", "告别", goodbyeTaskHandler).
		AddDependency("hello", "process").
		AddDependency("process", "goodbye").
		Build()

	if err != nil {
		return fmt.Errorf("构建工作流失败: %w", err)
	}

	// 3. 注册工作流
	if err := client.RegisterWorkflow(workflow); err != nil {
		return fmt.Errorf("注册工作流失败: %w", err)
	}

	// 4. 执行工作流
	input := map[string]interface{}{
		"name": "世界",
		"data": []int{1, 2, 3, 4, 5},
	}

	result, err := client.SimpleExecute("quick-start", input)
	if err != nil {
		return fmt.Errorf("执行工作流失败: %w", err)
	}

	fmt.Printf("执行完成: %s\n", result.ID)
	fmt.Printf("状态: %s\n", result.Status)
	fmt.Printf("结果: %+v\n", result.Output)

	// 5. 查看Web管理界面（如果启用）
	if client.IsWebEnabled() {
		fmt.Printf("Web管理界面: %s\n", client.GetWebURL())
	}

	return nil
}

// AdvancedExample 高级示例
func AdvancedExample() error {
	// 1. 创建自定义配置的客户端
	config := DefaultClientConfig()
	config.MySQLDSN = "user:pass@tcp(localhost:3306)/my_task_db?charset=utf8mb4&parseTime=True&loc=Local"
	config.WebPort = 9090
	config.MaxConcurrency = 20
	config.MaxAutoRetries = 5

	client, err := NewClient(config)
	if err != nil {
		return fmt.Errorf("创建客户端失败: %w", err)
	}
	defer client.Close()

	// 2. 创建复杂工作流
	builder := client.CreateWorkflow("data-pipeline")
	workflow, err := builder.
		SetDescription("数据处理管道").
		SetVersion("2.1.0").
		// 第一阶段：数据获取
		AddTask("fetch_data", "获取数据", fetchDataHandler).
		AddTask("validate_data", "验证数据", validateDataHandler).
		// 第二阶段：数据处理
		AddTask("clean_data", "清洗数据", cleanDataHandler).
		AddTask("transform_data", "转换数据", transformDataHandler).
		AddTask("aggregate_data", "聚合数据", aggregateDataHandler).
		// 第三阶段：数据存储
		AddTask("save_data", "保存数据", saveDataHandler).
		AddTask("create_report", "生成报告", createReportHandler).
		// 依赖关系
		AddDependency("fetch_data", "validate_data").
		AddDependency("validate_data", "clean_data").
		AddDependency("clean_data", "transform_data").
		AddDependency("clean_data", "aggregate_data").
		AddDependency("transform_data", "save_data").
		AddDependency("aggregate_data", "save_data").
		AddDependency("save_data", "create_report").
		Build()

	if err != nil {
		return fmt.Errorf("构建工作流失败: %w", err)
	}

	// 3. 注册工作流
	if err := client.RegisterWorkflow(workflow); err != nil {
		return fmt.Errorf("注册工作流失败: %w", err)
	}

	// 4. 异步执行工作流
	input := map[string]interface{}{
		"source":      "database",
		"table":       "user_events",
		"date_range":  "2024-01-01:2024-01-31",
		"output_path": "/tmp/reports/",
	}

	executionID, err := client.SimpleExecuteAsync("data-pipeline", input)
	if err != nil {
		return fmt.Errorf("执行工作流失败: %w", err)
	}

	fmt.Printf("异步执行已启动: %s\n", executionID)

	// 5. 监控执行进度
	for {
		status, err := client.GetExecutionStatus(executionID)
		if err != nil {
			log.Printf("获取执行状态失败: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		fmt.Printf("执行状态: %s, 进度: %.1f%% (%d/%d)\n",
			status.Status, status.Progress, status.CompletedTasks, status.TotalTasks)

		if status.Status == "success" || status.Status == "failed" {
			break
		}

		time.Sleep(5 * time.Second)
	}

	// 6. 获取最终结果
	result, err := client.GetExecution(executionID)
	if err != nil {
		return fmt.Errorf("获取执行结果失败: %w", err)
	}

	fmt.Printf("执行完成: %s\n", result.Status)
	if result.IsSuccess() {
		fmt.Printf("输出: %+v\n", result.Output)
	} else {
		fmt.Printf("执行失败，查看日志获取详细信息\n")
		
		// 获取执行日志
		logs, err := client.GetExecutionLogs(executionID, 50, 0)
		if err == nil {
			fmt.Println("最近的日志:")
			for _, logEntry := range logs {
				fmt.Printf("[%s] %s: %s\n", 
					logEntry.Level, logEntry.Timestamp.Format(time.RFC3339), logEntry.Message)
			}
		}
	}

	return nil
}

// RetryExample 重试示例
func RetryExample() error {
	client, err := NewSimpleClient()
	if err != nil {
		return fmt.Errorf("创建客户端失败: %w", err)
	}
	defer client.Close()

	// 1. 创建可能失败的工作流
	builder := client.CreateWorkflow("flaky-workflow")
	workflow, err := builder.
		SetDescription("可能失败的工作流示例").
		AddTask("flaky_task", "不稳定任务", flakyTaskHandler).
		AddTask("final_task", "最终任务", finalTaskHandler).
		AddDependency("flaky_task", "final_task").
		Build()

	if err != nil {
		return fmt.Errorf("构建工作流失败: %w", err)
	}

	if err := client.RegisterWorkflow(workflow); err != nil {
		return fmt.Errorf("注册工作流失败: %w", err)
	}

	// 2. 执行工作流（可能失败）
	input := map[string]interface{}{
		"failure_rate": 0.7, // 70%失败率
	}

	result, err := client.SimpleExecute("flaky-workflow", input)
	if err != nil {
		fmt.Printf("工作流执行失败: %v\n", err)
	} else if result.IsFailed() {
		fmt.Printf("工作流执行失败: %s\n", result.Status)
	} else {
		fmt.Printf("工作流执行成功: %s\n", result.Status)
		return nil // 成功了就返回
	}

	// 3. 检查失败的执行
	failedExecutions, err := client.GetFailedExecutions()
	if err != nil {
		return fmt.Errorf("获取失败执行失败: %w", err)
	}

	fmt.Printf("找到 %d 个失败的执行\n", len(failedExecutions))

	// 4. 手动重试
	for _, failed := range failedExecutions {
		if failed.WorkflowID == "flaky-workflow" {
			fmt.Printf("手动重试执行: %s\n", failed.ExecutionID)
			
			retryResult, err := client.ManualRetry(context.Background(), failed.ExecutionID)
			if err != nil {
				fmt.Printf("重试失败: %v\n", err)
				continue
			}

			fmt.Printf("重试结果: %s\n", retryResult.Status)
			if retryResult.IsSuccess() {
				fmt.Printf("重试成功! 输出: %+v\n", retryResult.Output)
			}
			break
		}
	}

	return nil
}

// MonitoringExample 监控示例
func MonitoringExample() error {
	client, err := NewSimpleClient()
	if err != nil {
		return fmt.Errorf("创建客户端失败: %w", err)
	}
	defer client.Close()

	// 1. 列出所有工作流
	workflows, err := client.ListWorkflows()
	if err != nil {
		return fmt.Errorf("列出工作流失败: %w", err)
	}

	fmt.Printf("共有 %d 个工作流:\n", len(workflows))
	for _, wf := range workflows {
		fmt.Printf("  - %s (%s) - %s\n", wf.Name, wf.Version, wf.Status)
	}

	// 2. 列出最近的执行
	executions, err := client.ListExecutions("", 10, 0)
	if err != nil {
		return fmt.Errorf("列出执行失败: %w", err)
	}

	fmt.Printf("\n最近 %d 次执行:\n", len(executions))
	for _, exec := range executions {
		fmt.Printf("  - %s: %s (%v)\n", 
			exec.ID[:8], exec.Status, exec.Duration)
	}

	// 3. 健康检查
	if err := client.HealthCheck(); err != nil {
		fmt.Printf("健康检查失败: %v\n", err)
	} else {
		fmt.Println("系统健康状态: 正常")
	}

	// 4. 显示Web管理界面信息
	if client.IsWebEnabled() {
		fmt.Printf("\nWeb管理界面可用: %s\n", client.GetWebURL())
		fmt.Println("您可以在浏览器中打开上述地址查看详细的监控信息")
	}

	return nil
}

// 示例任务处理器

func helloTaskHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	name := input["name"].(string)
	message := fmt.Sprintf("你好, %s!", name)
	
	return map[string]interface{}{
		"greeting": message,
		"timestamp": time.Now(),
	}, nil
}

func processTaskHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	// 获取上一个任务的输出
	helloOutput := input["hello_output"].(map[string]interface{})
	greeting := helloOutput["greeting"].(string)
	
	// 处理数据
	data := input["data"].([]interface{})
	var sum int
	for _, v := range data {
		if intVal, ok := v.(int); ok {
			sum += intVal
		} else if floatVal, ok := v.(float64); ok {
			sum += int(floatVal)
		}
	}
	
	return map[string]interface{}{
		"greeting": greeting,
		"sum": sum,
		"count": len(data),
	}, nil
}

func goodbyeTaskHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	processOutput := input["process_output"].(map[string]interface{})
	
	return map[string]interface{}{
		"message": "再见!",
		"summary": fmt.Sprintf("处理完成，总和: %v", processOutput["sum"]),
		"completed_at": time.Now(),
	}, nil
}

// 高级示例的任务处理器

func fetchDataHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	source := input["source"].(string)
	table := input["table"].(string)
	
	// 模拟数据获取
	time.Sleep(100 * time.Millisecond)
	
	return map[string]interface{}{
		"records": 1000,
		"source": source,
		"table": table,
		"fetched_at": time.Now(),
	}, nil
}

func validateDataHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	fetchOutput := input["fetch_data_output"].(map[string]interface{})
	records := fetchOutput["records"].(int)
	
	// 模拟数据验证
	time.Sleep(50 * time.Millisecond)
	
	validRecords := int(float64(records) * 0.95) // 95% 有效
	
	return map[string]interface{}{
		"total_records": records,
		"valid_records": validRecords,
		"invalid_records": records - validRecords,
		"validation_passed": true,
	}, nil
}

func cleanDataHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	validateOutput := input["validate_data_output"].(map[string]interface{})
	validRecords := validateOutput["valid_records"].(int)
	
	// 模拟数据清洗
	time.Sleep(200 * time.Millisecond)
	
	cleanedRecords := int(float64(validRecords) * 0.98) // 98% 清洗后保留
	
	return map[string]interface{}{
		"cleaned_records": cleanedRecords,
		"removed_records": validRecords - cleanedRecords,
	}, nil
}

func transformDataHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	cleanOutput := input["clean_data_output"].(map[string]interface{})
	cleanedRecords := cleanOutput["cleaned_records"].(int)
	
	// 模拟数据转换
	time.Sleep(300 * time.Millisecond)
	
	return map[string]interface{}{
		"transformed_records": cleanedRecords,
		"transformation_type": "normalize",
	}, nil
}

func aggregateDataHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	cleanOutput := input["clean_data_output"].(map[string]interface{})
	cleanedRecords := cleanOutput["cleaned_records"].(int)
	
	// 模拟数据聚合
	time.Sleep(150 * time.Millisecond)
	
	return map[string]interface{}{
		"aggregated_groups": 50,
		"base_records": cleanedRecords,
		"aggregation_type": "daily_summary",
	}, nil
}

func saveDataHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	transformOutput := input["transform_data_output"].(map[string]interface{})
	aggregateOutput := input["aggregate_data_output"].(map[string]interface{})
	
	// 模拟数据保存
	time.Sleep(100 * time.Millisecond)
	
	return map[string]interface{}{
		"saved_records": transformOutput["transformed_records"],
		"saved_aggregates": aggregateOutput["aggregated_groups"],
		"save_path": "/data/processed/",
		"saved_at": time.Now(),
	}, nil
}

func createReportHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	saveOutput := input["save_data_output"].(map[string]interface{})
	
	// 模拟报告生成
	time.Sleep(80 * time.Millisecond)
	
	return map[string]interface{}{
		"report_generated": true,
		"report_path": "/reports/daily_summary.pdf",
		"processed_records": saveOutput["saved_records"],
		"report_created_at": time.Now(),
	}, nil
}

func flakyTaskHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	failureRate := input["failure_rate"].(float64)
	
	// 模拟不稳定的任务
	time.Sleep(100 * time.Millisecond)
	
	if time.Now().UnixNano()%100 < int64(failureRate*100) {
		return nil, fmt.Errorf("任务随机失败 (失败率: %.0f%%)", failureRate*100)
	}
	
	return map[string]interface{}{
		"result": "success",
		"attempts": 1,
	}, nil
}

func finalTaskHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	flakyOutput := input["flaky_task_output"].(map[string]interface{})
	
	return map[string]interface{}{
		"final_result": "completed",
		"previous_result": flakyOutput["result"],
		"completed_at": time.Now(),
	}, nil
}
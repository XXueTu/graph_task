package graph_task

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/XXueTu/graph_task/client"
	"github.com/XXueTu/graph_task/engine"
	"github.com/XXueTu/graph_task/event"
	"github.com/XXueTu/graph_task/storage"
)

const (
	mysqlDSN = "root:Root@123@tcp(10.99.51.9:3306)/graph_task?charset=utf8mb4&parseTime=True&loc=Local"
)

// TestBasicWorkflow 基础工作流测试
func TestBasicWorkflow(t *testing.T) {
	// 创建客户端
	storage, err := storage.NewMySQLExecutionStorage(mysqlDSN)
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}

	client := client.NewClient(engine.WithStorage(storage))
	defer client.Close()

	// 创建工作流
	workflow, err := client.CreateWorkflow("basic_example").
		SetDescription("基础示例工作流").
		SetVersion("1.0.0").
		AddSimpleTask("task1", "初始化", func() error {
			fmt.Println("执行任务1: 初始化")
			time.Sleep(100 * time.Millisecond)
			return nil
		}).
		AddSimpleTask("task2", "处理数据", func() error {
			fmt.Println("执行任务2: 处理数据")
			time.Sleep(200 * time.Millisecond)
			return nil
		}).
		AddSimpleTask("task3", "清理资源", func() error {
			fmt.Println("执行任务3: 清理资源")
			time.Sleep(100 * time.Millisecond)
			return nil
		}).
		AddDependency("task1", "task2").
		AddDependency("task2", "task3").
		SetTaskTimeout("task2", 5).
		SetTaskRetry("task2", 2).
		Publish()

	if err != nil {
		t.Fatalf("创建工作流失败: %v", err)
	}

	// 执行工作流
	ctx := context.Background()
	result, err := client.Execute(ctx, workflow.ID, map[string]any{
		"input_data": "test_data",
	})

	if err != nil {
		t.Fatalf("执行工作流失败: %v", err)
	}

	if result.Status != engine.WorkflowStatusSuccess {
		t.Fatalf("工作流执行失败: %s", result.Error)
	}

	fmt.Printf("工作流执行成功，耗时: %v\n", result.Duration)
}

// TestDataPipeline 数据管道测试
func TestDataPipeline(t *testing.T) {
	storage, err := storage.NewMySQLExecutionStorage(mysqlDSN)
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}
	client := client.NewClient(engine.WithStorage(storage))
	defer client.Close()

	_ = client.Subscribe(event.EventWorkflowPublished, event.EventWorkflowPublishedHandler)

	_ = client.Subscribe(event.EventWorkflowCompleted, event.EventWorkflowCompletedHandler)

	_ = client.Subscribe(event.EventTracePublished, event.EventTracePublishedHandler)

	// 创建数据处理管道
	workflow, err := client.DataPipeline("data_pipeline",
		// 阶段1：数据清洗
		func(input map[string]any) (map[string]any, error) {
			fmt.Println("阶段1: 数据清洗")
			data := input["raw_data"].(string)
			cleaned := fmt.Sprintf("cleaned_%s", data)
			return map[string]any{"cleaned_data": cleaned}, nil
		},
		// 阶段2：数据转换
		func(input map[string]any) (map[string]any, error) {
			fmt.Println("阶段2: 数据转换")
			cleaned := input["stage_1_cleaned_data"].(string)
			transformed := fmt.Sprintf("transformed_%s", cleaned)
			return map[string]any{"transformed_data": transformed}, nil
		},
		// 阶段3：数据输出
		func(input map[string]any) (map[string]any, error) {
			fmt.Println("阶段3: 数据输出")
			transformed := input["stage_2_transformed_data"].(string)
			output := fmt.Sprintf("output_%s", transformed)
			return map[string]any{"final_output": output}, nil
		},
	)

	if err != nil {
		t.Fatalf("创建数据管道失败: %v", err)
	}

	// 执行数据管道
	result, err := client.Execute(context.Background(), workflow.ID, map[string]any{
		"raw_data": "user_input",
	})

	if err != nil {
		t.Fatalf("执行数据管道失败: %v", err)
	}

	if result.Status != engine.WorkflowStatusSuccess {
		t.Fatalf("数据管道执行失败: %s", result.Error)
	}

	fmt.Printf("数据管道执行成功，最终输出: %v\n", result.Output)
	time.Sleep(10 * time.Second)
}

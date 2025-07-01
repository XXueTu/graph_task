package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/XXueTu/graph_task/application"
	"github.com/XXueTu/graph_task/domain/execution"
	"github.com/XXueTu/graph_task/domain/retry"
	"github.com/XXueTu/graph_task/domain/workflow"
)

const mysqlDSN = "root:Root@123@tcp(10.99.51.9:3306)/graph_task?charset=utf8mb4&parseTime=True&loc=Local"

func TestEngineBasicWorkflow(t *testing.T) {
	// 创建测试引擎
	config := DefaultEngineConfig()
	config.MySQLDSN = mysqlDSN

	engine, err := NewEngine(config)
	if err != nil {
		t.Skipf("跳过测试，无法连接数据库: %v", err)
		return
	}
	defer engine.Close()

	// 创建简单工作流
	builder := engine.CreateWorkflow("test-workflow")
	workflow, err := builder.
		SetDescription("测试工作流").
		AddTask("task1", "任务1", simpleTask).
		AddTask("task2", "任务2", simpleTask).
		AddDependency("task1", "task2").
		Build()

	if err != nil {
		t.Fatalf("构建工作流失败: %v", err)
	}

	// 发布工作流
	if err := engine.PublishWorkflow(workflow); err != nil {
		t.Fatalf("发布工作流失败: %v", err)
	}

	// 执行工作流
	ctx := context.Background()
	input := map[string]interface{}{"test": "data"}

	execution, err := engine.Execute(ctx, workflow.ID(), input)
	if err != nil {
		t.Fatalf("执行工作流失败: %v", err)
	}

	// 验证执行结果
	if execution.Status().String() != "success" {
		t.Errorf("期望执行状态为success，实际为: %s", execution.Status())
	}

	if len(execution.TaskResults()) != 2 {
		t.Errorf("期望2个任务结果，实际为: %d", len(execution.TaskResults()))
	}
	time.Sleep(10 * time.Second)
}

func TestEngineAsyncExecution(t *testing.T) {
	config := DefaultEngineConfig()
	config.MySQLDSN = "root:password@tcp(localhost:3306)/task_graph_test"

	engine, err := NewEngine(config)
	if err != nil {
		t.Skipf("跳过测试，无法连接数据库: %v", err)
		return
	}
	defer engine.Close()

	// 创建工作流
	builder := engine.CreateWorkflow("async-test-workflow")
	workflow, err := builder.
		SetDescription("异步测试工作流").
		AddTask("async-task", "异步任务", slowTask).
		Build()

	if err != nil {
		t.Fatalf("构建工作流失败: %v", err)
	}

	if err := engine.PublishWorkflow(workflow); err != nil {
		t.Fatalf("发布工作流失败: %v", err)
	}

	// 异步执行
	ctx := context.Background()
	input := map[string]interface{}{"delay": 100}

	executionID, err := engine.ExecuteAsync(ctx, workflow.ID(), input)
	if err != nil {
		t.Fatalf("异步执行失败: %v", err)
	}

	// 等待执行完成
	var execution *execution.Execution
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		execution, err = engine.GetExecution(executionID)
		if err != nil {
			continue
		}
		if execution.IsCompleted() {
			break
		}
	}

	if execution == nil {
		t.Fatal("执行超时")
	}

	if execution.Status().String() != "success" {
		t.Errorf("期望执行状态为success，实际为: %s", execution.Status())
	}
}

func TestEngineRetryMechanism(t *testing.T) {
	config := DefaultEngineConfig()
	config.MySQLDSN = "root:password@tcp(localhost:3306)/task_graph_test"
	config.RetryConfig = retry.NewConfig(2, 100*time.Millisecond, 1.5, time.Second)

	engine, err := NewEngine(config)
	if err != nil {
		t.Skipf("跳过测试，无法连接数据库: %v", err)
		return
	}
	defer engine.Close()

	// 创建包含失败任务的工作流
	builder := engine.CreateWorkflow("retry-test-workflow")
	workflow, err := builder.
		SetDescription("重试测试工作流").
		AddTask("failing-task", "失败任务", failingTask).
		Build()

	if err != nil {
		t.Fatalf("构建工作流失败: %v", err)
	}

	if err := engine.PublishWorkflow(workflow); err != nil {
		t.Fatalf("发布工作流失败: %v", err)
	}

	// 执行工作流（应该失败）
	ctx := context.Background()
	input := map[string]interface{}{"should_fail": true}

	execution, err := engine.Execute(ctx, workflow.ID(), input)
	if err == nil {
		t.Error("期望执行失败，但成功了")
	}

	if execution.Status().String() != "failed" {
		t.Errorf("期望执行状态为failed，实际为: %s", execution.Status())
	}

	// 检查重试统计
	stats, err := engine.GetRetryStatistics()
	if err != nil {
		t.Fatalf("获取重试统计失败: %v", err)
	}

	if stats.TotalFailed() == 0 {
		t.Error("期望有失败的执行记录")
	}
}

func TestEngineWorkflowManagement(t *testing.T) {
	config := DefaultEngineConfig()
	config.MySQLDSN = "root:password@tcp(localhost:3306)/task_graph_test"

	engine, err := NewEngine(config)
	if err != nil {
		t.Skipf("跳过测试，无法连接数据库: %v", err)
		return
	}
	defer engine.Close()

	// 创建多个工作流
	workflows := []string{"workflow1", "workflow2", "workflow3"}

	for _, name := range workflows {
		builder := engine.CreateWorkflow(name)
		workflow, err := builder.
			SetDescription("测试工作流: "+name).
			AddTask("task1", "任务1", simpleTask).
			Build()

		if err != nil {
			t.Fatalf("构建工作流%s失败: %v", name, err)
		}

		if err := engine.PublishWorkflow(workflow); err != nil {
			t.Fatalf("发布工作流%s失败: %v", name, err)
		}
	}

	// 列出工作流
	list, err := engine.ListWorkflows()
	if err != nil {
		t.Fatalf("列出工作流失败: %v", err)
	}

	if len(list) < len(workflows) {
		t.Errorf("期望至少%d个工作流，实际为: %d", len(workflows), len(list))
	}

	// 获取特定工作流
	for _, wf := range list {
		retrieved, err := engine.GetWorkflow(wf.ID())
		if err != nil {
			t.Errorf("获取工作流%s失败: %v", wf.ID(), err)
		}

		if retrieved.ID() != wf.ID() {
			t.Errorf("工作流ID不匹配: 期望%s，实际%s", wf.ID(), retrieved.ID())
		}
	}
}

func TestEngineEventSubscription(t *testing.T) {
	config := DefaultEngineConfig()
	config.MySQLDSN = "root:password@tcp(localhost:3306)/task_graph_test"

	engine, err := NewEngine(config)
	if err != nil {
		t.Skipf("跳过测试，无法连接数据库: %v", err)
		return
	}
	defer engine.Close()

	// 订阅事件
	eventCh := make(chan string, 10)
	handler := func(event *application.Event) error {
		eventCh <- event.Type()
		return nil
	}

	if err := engine.Subscribe("workflow.published", handler); err != nil {
		t.Fatalf("订阅事件失败: %v", err)
	}

	// 发布工作流，触发事件
	builder := engine.CreateWorkflow("event-test-workflow")
	workflow, err := builder.
		SetDescription("事件测试工作流").
		AddTask("task1", "任务1", simpleTask).
		Build()

	if err != nil {
		t.Fatalf("构建工作流失败: %v", err)
	}

	if err := engine.PublishWorkflow(workflow); err != nil {
		t.Fatalf("发布工作流失败: %v", err)
	}

	// 检查是否收到事件
	select {
	case eventType := <-eventCh:
		if eventType != "workflow.published" {
			t.Errorf("期望收到workflow.published事件，实际收到: %s", eventType)
		}
	case <-time.After(time.Second):
		t.Error("没有收到期望的事件")
	}
}

// 测试任务函数
func simpleTask(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	ctx.Logger.Info(ctx.Context, "simpleTask info", map[string]interface{}{
		"input": input,
	})
	ctx.Logger.Debug(ctx.Context, "simpleTask debug", map[string]interface{}{
		"input": input,
	})
	ctx.Logger.Warn(ctx.Context, "simpleTask warn", map[string]interface{}{
		"input": input,
	})
	ctx.Logger.Error(ctx.Context, "simpleTask error", map[string]interface{}{
		"input": input,
	})
	return map[string]interface{}{
		"result": "success",
		"input":  input,
	}, nil
}

func slowTask(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	delay, ok := input["delay"].(int)
	if !ok {
		delay = 100
	}

	time.Sleep(time.Duration(delay) * time.Millisecond)

	return map[string]interface{}{
		"result": "completed after delay",
		"delay":  delay,
	}, nil
}

func failingTask(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
	shouldFail, ok := input["should_fail"].(bool)
	if ok && shouldFail {
		return nil, fmt.Errorf("任务故意失败")
	}

	return map[string]interface{}{
		"result": "success",
	}, nil
}

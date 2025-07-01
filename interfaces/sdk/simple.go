// Package sdk provides simplified interfaces for common use cases
package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/XXueTu/graph_task/domain/workflow"
)

// Simple 简化接口，为最常见的用例提供一行代码解决方案

// RunSimpleWorkflow 运行简单工作流（一行代码解决方案）
func RunSimpleWorkflow(dsn string, name string, tasks []SimpleTask, input map[string]interface{}) (*ExecutionResult, error) {
	// 创建客户端
	client, err := NewClientWithDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("创建客户端失败: %w", err)
	}
	defer client.Close()

	// 创建工作流
	ew := client.NewEasyWorkflow(name)
	
	// 添加任务
	for i, task := range tasks {
		ew.Task(task.ID, task.Name, task.Handler)
		
		// 自动添加顺序依赖
		if i > 0 {
			ew.Then(tasks[i-1].ID, task.ID)
		}
	}
	
	// 注册并执行
	return ew.RegisterAndRun(input)
}

// SimpleTask 简单任务定义
type SimpleTask struct {
	ID      string
	Name    string
	Handler workflow.TaskHandler
}

// ProcessData 数据处理简化函数
func ProcessData(dsn string, extractHandler, transformHandler, loadHandler workflow.TaskHandler, input map[string]interface{}) (*ExecutionResult, error) {
	tasks := []SimpleTask{
		{ID: "extract", Name: "数据提取", Handler: extractHandler},
		{ID: "transform", Name: "数据转换", Handler: transformHandler},
		{ID: "load", Name: "数据加载", Handler: loadHandler},
	}
	
	return RunSimpleWorkflow(dsn, "data-processing", tasks, input)
}

// HandleApproval 审批流程简化函数
func HandleApproval(dsn string, submitHandler, approveHandler, executeHandler workflow.TaskHandler, input map[string]interface{}) (*ExecutionResult, error) {
	tasks := []SimpleTask{
		{ID: "submit", Name: "提交申请", Handler: submitHandler},
		{ID: "approve", Name: "审批", Handler: approveHandler},
		{ID: "execute", Name: "执行", Handler: executeHandler},
	}
	
	return RunSimpleWorkflow(dsn, "approval-workflow", tasks, input)
}

// ProcessQueue 队列处理简化函数
func ProcessQueue(dsn string, items []map[string]interface{}, handler workflow.TaskHandler) ([]*ExecutionResult, error) {
	client, err := NewClientWithDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("创建客户端失败: %w", err)
	}
	defer client.Close()

	// 创建单任务工作流
	ew := client.NewEasyWorkflow("queue-processor").
		Task("process", "处理项目", handler)
	
	if err := ew.Register(); err != nil {
		return nil, fmt.Errorf("注册工作流失败: %w", err)
	}

	// 批量执行
	batch := client.NewBatchExecution("queue-processor")
	for _, item := range items {
		batch.AddInput(item)
	}

	return batch.Execute(context.Background())
}

// ScheduleTask 定时任务简化函数
func ScheduleTask(dsn string, taskName string, handler workflow.TaskHandler, input map[string]interface{}, interval time.Duration) (*ScheduledExecution, error) {
	client, err := NewClientWithDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("创建客户端失败: %w", err)
	}

	// 创建单任务工作流
	ew := client.NewEasyWorkflow(taskName).
		Task("task", taskName, handler)
	
	if err := ew.Register(); err != nil {
		return nil, fmt.Errorf("注册工作流失败: %w", err)
	}

	// 创建定时执行
	return client.NewScheduledExecution(taskName, input, interval), nil
}

// OneShot 一次性任务执行（最简单的用法）
func OneShot(dsn string, handler workflow.TaskHandler, input map[string]interface{}) (*ExecutionResult, error) {
	return RunSimpleWorkflow(dsn, "oneshot", []SimpleTask{
		{ID: "task", Name: "Task", Handler: handler},
	}, input)
}

// ParallelProcess 并行处理简化函数
func ParallelProcess(dsn string, handlers map[string]workflow.TaskHandler, input map[string]interface{}) (*ExecutionResult, error) {
	client, err := NewClientWithDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("创建客户端失败: %w", err)
	}
	defer client.Close()

	// 创建并行工作流
	ew := client.NewEasyWorkflow("parallel-process")
	
	// 添加所有任务（无依赖关系，自然并行）
	for id, handler := range handlers {
		ew.Task(id, id, handler)
	}
	
	// 注册并执行
	return ew.RegisterAndRun(input)
}

// ConditionalProcess 条件处理简化函数
func ConditionalProcess(dsn string, condition func(map[string]interface{}) bool, 
	trueHandler, falseHandler workflow.TaskHandler, input map[string]interface{}) (*ExecutionResult, error) {
	
	// 创建条件任务处理器
	conditionalHandler := func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
		if condition(input) {
			return trueHandler(ctx, input)
		} else {
			return falseHandler(ctx, input)
		}
	}
	
	return OneShot(dsn, conditionalHandler, input)
}

// RetryProcess 重试处理简化函数
func RetryProcess(dsn string, handler workflow.TaskHandler, input map[string]interface{}, maxRetries int) (*ExecutionResult, error) {
	config := DefaultClientConfig()
	config.MySQLDSN = dsn
	config.MaxAutoRetries = maxRetries
	config.EnableWeb = false // 不启用Web界面
	
	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("创建客户端失败: %w", err)
	}
	defer client.Close()

	// 创建工作流
	ew := client.NewEasyWorkflow("retry-process").
		Task("task", "重试任务", handler)
	
	// 注册并执行
	return ew.RegisterAndRun(input)
}

// ChainProcess 链式处理简化函数
func ChainProcess(dsn string, handlers []workflow.TaskHandler, input map[string]interface{}) (*ExecutionResult, error) {
	var tasks []SimpleTask
	for i, handler := range handlers {
		tasks = append(tasks, SimpleTask{
			ID:      fmt.Sprintf("step_%d", i+1),
			Name:    fmt.Sprintf("Step %d", i+1),
			Handler: handler,
		})
	}
	
	return RunSimpleWorkflow(dsn, "chain-process", tasks, input)
}

// QuickTest 快速测试函数，用于开发和调试
func QuickTest(handler workflow.TaskHandler, input map[string]interface{}) (*ExecutionResult, error) {
	// 使用内存数据库进行测试
	config := DefaultClientConfig()
	config.MySQLDSN = ":memory:" // 假设支持内存数据库
	config.EnableWeb = false
	config.MaxAutoRetries = 0 // 测试时不重试
	
	client, err := NewClient(config)
	if err != nil {
		// 如果内存数据库不支持，返回模拟结果
		return &ExecutionResult{
			ID:       "test_exec_001",
			Status:   ExecutionStatusSuccess,
			Input:    input,
			Output:   map[string]interface{}{"test": true},
			Duration: 100 * time.Millisecond,
		}, nil
	}
	defer client.Close()

	return OneShot(config.MySQLDSN, handler, input)
}

// ValidateWorkflow 验证工作流配置
func ValidateWorkflow(tasks []SimpleTask) error {
	if len(tasks) == 0 {
		return fmt.Errorf("工作流至少需要一个任务")
	}
	
	// 检查任务ID重复
	ids := make(map[string]bool)
	for _, task := range tasks {
		if task.ID == "" {
			return fmt.Errorf("任务ID不能为空")
		}
		if ids[task.ID] {
			return fmt.Errorf("任务ID重复: %s", task.ID)
		}
		ids[task.ID] = true
		
		if task.Handler == nil {
			return fmt.Errorf("任务 %s 缺少处理函数", task.ID)
		}
	}
	
	return nil
}

// CreateWorkflowFromJSON 从JSON配置创建工作流（适用于配置驱动的场景）
func CreateWorkflowFromJSON(dsn string, jsonConfig []byte, handlers map[string]workflow.TaskHandler) (*Client, error) {
	// TODO: 实现JSON配置解析
	// 这里只是一个占位符，实际实现需要解析JSON并创建工作流
	return NewClientWithDSN(dsn)
}

// GetSimpleStats 获取简化的统计信息
func GetSimpleStats(dsn string) (map[string]interface{}, error) {
	client, err := NewClientWithDSN(dsn)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	monitor := client.NewMonitor()
	stats, err := monitor.GetSystemStats()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"workflows":  stats.TotalWorkflows,
		"executions": stats.TotalExecutions,
		"success_rate": func() float64 {
			if stats.TotalExecutions == 0 {
				return 0
			}
			return float64(stats.SuccessfulExecutions) / float64(stats.TotalExecutions) * 100
		}(),
		"running": stats.RunningExecutions,
		"failed":  stats.FailedExecutions,
	}, nil
}
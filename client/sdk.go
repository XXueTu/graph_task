package client

import (
	"context"
	"fmt"
	"time"

	"github.com/XXueTu/graph_task/engine"
	"github.com/XXueTu/graph_task/event"
)

// Client SDK客户端
type Client struct {
	engine engine.Engine
}

// NewClient 创建SDK客户端
func NewClient(opts ...engine.EngineOption) *Client {
	return &Client{
		engine: engine.NewEngine(opts...),
	}
}

// CreateWorkflow 创建工作流
func (c *Client) CreateWorkflow(name string) *SDKWorkflowBuilder {
	return &SDKWorkflowBuilder{
		builder: c.engine.CreateWorkflow(name),
		client:  c,
	}
}

// SDKWorkflowBuilder SDK工作流构建器
type SDKWorkflowBuilder struct {
	builder engine.WorkflowBuilder
	client  *Client
}

// SetDescription 设置描述
func (wb *SDKWorkflowBuilder) SetDescription(description string) *SDKWorkflowBuilder {
	wb.builder.SetDescription(description)
	return wb
}

// SetVersion 设置版本
func (wb *SDKWorkflowBuilder) SetVersion(version string) *SDKWorkflowBuilder {
	wb.builder.SetVersion(version)
	return wb
}

// AddTask 添加任务
func (wb *SDKWorkflowBuilder) AddTask(taskID, name string, handler engine.TaskHandler) *SDKWorkflowBuilder {
	wb.builder.AddTask(taskID, name, handler)
	return wb
}

// AddSimpleTask 添加简单任务（无输入输出）
func (wb *SDKWorkflowBuilder) AddSimpleTask(taskID, name string, fn func() error) *SDKWorkflowBuilder {
	handler := func(ctx context.Context, input map[string]any) (map[string]any, error) {
		err := fn()
		return make(map[string]any), err
	}
	wb.builder.AddTask(taskID, name, handler)
	return wb
}

// AddDataTask 添加数据处理任务
func (wb *SDKWorkflowBuilder) AddDataTask(taskID, name string, fn func(input map[string]any) (map[string]any, error)) *SDKWorkflowBuilder {
	handler := func(ctx context.Context, input map[string]any) (map[string]any, error) {
		return fn(input)
	}
	wb.builder.AddTask(taskID, name, handler)
	return wb
}

// AddAsyncTask 添加异步任务
func (wb *SDKWorkflowBuilder) AddAsyncTask(taskID, name string, fn func(ctx context.Context, input map[string]any) (map[string]any, error)) *SDKWorkflowBuilder {
	wb.builder.AddTask(taskID, name, fn)
	return wb
}

// AddDependency 添加依赖
func (wb *SDKWorkflowBuilder) AddDependency(fromTask, toTask string) *SDKWorkflowBuilder {
	wb.builder.AddDependency(fromTask, toTask)
	return wb
}

// SetTaskTimeout 设置任务超时
func (wb *SDKWorkflowBuilder) SetTaskTimeout(taskID string, timeoutSeconds int) *SDKWorkflowBuilder {
	wb.builder.SetTaskTimeout(taskID, timeoutSeconds)
	return wb
}

// SetTaskRetry 设置任务重试
func (wb *SDKWorkflowBuilder) SetTaskRetry(taskID string, retry int) *SDKWorkflowBuilder {
	wb.builder.SetTaskRetry(taskID, retry)
	return wb
}

// SetTaskInput 设置任务输入
func (wb *SDKWorkflowBuilder) SetTaskInput(taskID string, input map[string]any) *SDKWorkflowBuilder {
	wb.builder.SetTaskInput(taskID, input)
	return wb
}

// Publish 发布工作流
func (wb *SDKWorkflowBuilder) Publish() (*engine.Workflow, error) {
	workflow, err := wb.builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build workflow: %w", err)
	}

	if err := wb.client.engine.PublishWorkflow(workflow); err != nil {
		return nil, fmt.Errorf("failed to publish workflow: %w", err)
	}

	return workflow, nil
}

// Execute 执行工作流
func (c *Client) Execute(ctx context.Context, workflowID string, input map[string]any) (*engine.ExecutionResult, error) {
	return c.engine.Execute(ctx, workflowID, input)
}

// ExecuteAsync 异步执行工作流
func (c *Client) ExecuteAsync(ctx context.Context, workflowID string, input map[string]any) (string, error) {
	return c.engine.ExecuteAsync(ctx, workflowID, input)
}

// GetExecutionResult 获取执行结果
func (c *Client) GetExecutionResult(executionID string) (*engine.ExecutionResult, error) {
	return c.engine.GetExecutionResult(executionID)
}

// WaitForCompletion 等待执行完成
func (c *Client) WaitForCompletion(executionID string, timeout time.Duration) (*engine.ExecutionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for execution completion")
		case <-ticker.C:
			result, err := c.GetExecutionResult(executionID)
			if err != nil {
				continue
			}

			if result.Status != engine.WorkflowStatusRunning {
				return result, nil
			}
		}
	}
}

// GetWorkflow 获取工作流
func (c *Client) GetWorkflow(id string) (*engine.Workflow, error) {
	return c.engine.GetWorkflow(id)
}

// ListWorkflows 列出工作流
func (c *Client) ListWorkflows() ([]*engine.Workflow, error) {
	return c.engine.ListWorkflows()
}

// DeleteWorkflow 删除工作流
func (c *Client) DeleteWorkflow(id string) error {
	return c.engine.DeleteWorkflow(id)
}

// Close 关闭客户端
func (c *Client) Close() error {
	return c.engine.Close()
}

// Subscribe 订阅事件
func (c *Client) Subscribe(eventType string, handler event.EventHandler) error {
	return c.engine.Subscribe(eventType, handler)
}

// 快捷方法

// SimpleWorkflow 创建简单工作流的快捷方法
func (c *Client) SimpleWorkflow(name string, tasks ...func() error) (*engine.Workflow, error) {
	wb := c.CreateWorkflow(name)

	for i, task := range tasks {
		taskID := fmt.Sprintf("task_%d", i+1)
		taskName := fmt.Sprintf("Task %d", i+1)
		wb.AddSimpleTask(taskID, taskName, task)

		// 添加顺序依赖
		if i > 0 {
			prevTaskID := fmt.Sprintf("task_%d", i)
			wb.AddDependency(prevTaskID, taskID)
		}
	}

	return wb.Publish()
}

// ParallelWorkflow 创建并行工作流的快捷方法
func (c *Client) ParallelWorkflow(name string, tasks ...func() error) (*engine.Workflow, error) {
	wb := c.CreateWorkflow(name)

	for i, task := range tasks {
		taskID := fmt.Sprintf("task_%d", i+1)
		taskName := fmt.Sprintf("Task %d", i+1)
		wb.AddSimpleTask(taskID, taskName, task)
		// 并行执行，不添加依赖关系
	}

	return wb.Publish()
}

// DataPipeline 创建数据处理管道的快捷方法
func (c *Client) DataPipeline(name string, stages ...func(map[string]any) (map[string]any, error)) (*engine.Workflow, error) {
	wb := c.CreateWorkflow(name)

	for i, stage := range stages {
		taskID := fmt.Sprintf("stage_%d", i+1)
		taskName := fmt.Sprintf("Stage %d", i+1)
		wb.AddDataTask(taskID, taskName, stage)

		// 添加顺序依赖
		if i > 0 {
			prevTaskID := fmt.Sprintf("stage_%d", i)
			wb.AddDependency(prevTaskID, taskID)
		}
	}

	return wb.Publish()
}

// ConditionalWorkflow 条件工作流构建器
type ConditionalWorkflow struct {
	client    *Client
	workflow  *SDKWorkflowBuilder
	taskCount int
}

// NewConditionalWorkflow 创建条件工作流
func (c *Client) NewConditionalWorkflow(name string) *ConditionalWorkflow {
	return &ConditionalWorkflow{
		client:   c,
		workflow: c.CreateWorkflow(name),
	}
}

// AddConditionalTask 添加条件任务
func (cw *ConditionalWorkflow) AddConditionalTask(
	name string,
	condition func(map[string]any) bool,
	trueTask func(map[string]any) (map[string]any, error),
	falseTask func(map[string]any) (map[string]any, error),
) *ConditionalWorkflow {
	cw.taskCount++
	taskID := fmt.Sprintf("conditional_%d", cw.taskCount)

	handler := func(ctx context.Context, input map[string]any) (map[string]any, error) {
		if condition(input) {
			return trueTask(input)
		} else {
			return falseTask(input)
		}
	}

	cw.workflow.AddAsyncTask(taskID, name, handler)
	return cw
}

// Publish 发布条件工作流
func (cw *ConditionalWorkflow) Publish() (*engine.Workflow, error) {
	return cw.workflow.Publish()
}

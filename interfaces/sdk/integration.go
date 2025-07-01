package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/XXueTu/graph_task/domain/workflow"
)

// Integration SDK集成工具包，提供简化的集成方式

// QuickStart 快速启动函数，一行代码启动带Web界面的任务引擎
func QuickStart(dsn string, port int) (*Client, error) {
	config := &ClientConfig{
		MySQLDSN:          dsn,
		WebPort:           port,
		EnableWeb:         true,
		MaxConcurrency:    10,
		DefaultTimeout:    30 * time.Second,
		MaxAutoRetries:    3,
		RetryDelay:        1 * time.Second,
		BackoffMultiplier: 2.0,
		MaxRetryDelay:     30 * time.Second,
		LogLevel:          "info",
	}

	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("快速启动失败: %w", err)
	}

	fmt.Printf("🚀 任务引擎已启动，Web管理界面: %s\n", client.GetWebURL())
	return client, nil
}

// EasyWorkflow 简化的工作流创建助手
type EasyWorkflow struct {
	client *Client
	name   string
	tasks  []easyTask
	deps   []dependency
}

type easyTask struct {
	id      string
	name    string
	handler workflow.TaskHandler
	timeout time.Duration
}

type dependency struct {
	from, to string
}

// NewEasyWorkflow 创建简化工作流构建器
func (c *Client) NewEasyWorkflow(name string) *EasyWorkflow {
	return &EasyWorkflow{
		client: c,
		name:   name,
		tasks:  make([]easyTask, 0),
		deps:   make([]dependency, 0),
	}
}

// Task 添加任务
func (ew *EasyWorkflow) Task(id, name string, handler workflow.TaskHandler) *EasyWorkflow {
	ew.tasks = append(ew.tasks, easyTask{
		id:      id,
		name:    name,
		handler: handler,
		timeout: 30 * time.Second, // 默认超时
	})
	return ew
}

// TaskWithTimeout 添加带超时的任务
func (ew *EasyWorkflow) TaskWithTimeout(id, name string, handler workflow.TaskHandler, timeout time.Duration) *EasyWorkflow {
	ew.tasks = append(ew.tasks, easyTask{
		id:      id,
		name:    name,
		handler: handler,
		timeout: timeout,
	})
	return ew
}

// Then 添加依赖关系（A then B 表示A完成后执行B）
func (ew *EasyWorkflow) Then(from, to string) *EasyWorkflow {
	ew.deps = append(ew.deps, dependency{from: from, to: to})
	return ew
}

// Parallel 并行任务组（多个任务可以并行执行）
func (ew *EasyWorkflow) Parallel(taskIDs ...string) *EasyWorkflow {
	// 并行任务不需要添加依赖关系
	return ew
}

// Sequential 顺序任务组（按顺序执行）
func (ew *EasyWorkflow) Sequential(taskIDs ...string) *EasyWorkflow {
	for i := 0; i < len(taskIDs)-1; i++ {
		ew.Then(taskIDs[i], taskIDs[i+1])
	}
	return ew
}

// Register 注册工作流
func (ew *EasyWorkflow) Register() error {
	builder := ew.client.CreateWorkflow(ew.name)
	
	// 添加所有任务
	for _, task := range ew.tasks {
		builder.AddTask(task.id, task.name, task.handler)
	}
	
	// 添加所有依赖
	for _, dep := range ew.deps {
		builder.AddDependency(dep.from, dep.to)
	}
	
	// 构建并注册
	workflow, err := builder.Build()
	if err != nil {
		return fmt.Errorf("构建工作流失败: %w", err)
	}
	
	return ew.client.RegisterWorkflow(workflow)
}

// RegisterAndRun 注册并立即执行工作流
func (ew *EasyWorkflow) RegisterAndRun(input map[string]interface{}) (*ExecutionResult, error) {
	if err := ew.Register(); err != nil {
		return nil, err
	}
	
	return ew.client.SimpleExecute(ew.name, input)
}

// TemplateStep 模板步骤
type TemplateStep struct {
	ID           string
	Name         string
	Description  string
	Dependencies []string
}

// 预定义的工作流模板

// DataProcessingTemplate 数据处理模板
func DataProcessingTemplate() *WorkflowTemplate {
	return &WorkflowTemplate{
		Name:        "data-processing",
		Description: "标准数据处理流水线",
		Steps: []TemplateStep{
			{ID: "extract", Name: "数据提取", Description: "从数据源提取数据"},
			{ID: "validate", Name: "数据验证", Description: "验证数据质量", Dependencies: []string{"extract"}},
			{ID: "transform", Name: "数据转换", Description: "转换数据格式", Dependencies: []string{"validate"}},
			{ID: "load", Name: "数据加载", Description: "加载到目标系统", Dependencies: []string{"transform"}},
		},
	}
}

// MLPipelineTemplate 机器学习流水线模板
func MLPipelineTemplate() *WorkflowTemplate {
	return &WorkflowTemplate{
		Name:        "ml-pipeline",
		Description: "机器学习训练流水线",
		Steps: []TemplateStep{
			{ID: "prepare_data", Name: "数据准备", Description: "准备训练数据"},
			{ID: "feature_engineering", Name: "特征工程", Description: "特征提取和处理", Dependencies: []string{"prepare_data"}},
			{ID: "train_model", Name: "模型训练", Description: "训练机器学习模型", Dependencies: []string{"feature_engineering"}},
			{ID: "evaluate_model", Name: "模型评估", Description: "评估模型性能", Dependencies: []string{"train_model"}},
			{ID: "deploy_model", Name: "模型部署", Description: "部署模型到生产环境", Dependencies: []string{"evaluate_model"}},
		},
	}
}

// ApprovalWorkflowTemplate 审批流程模板
func ApprovalWorkflowTemplate() *WorkflowTemplate {
	return &WorkflowTemplate{
		Name:        "approval-workflow",
		Description: "多级审批流程",
		Steps: []TemplateStep{
			{ID: "submit_request", Name: "提交申请", Description: "提交审批申请"},
			{ID: "first_level_approval", Name: "一级审批", Description: "部门经理审批", Dependencies: []string{"submit_request"}},
			{ID: "second_level_approval", Name: "二级审批", Description: "总监审批", Dependencies: []string{"first_level_approval"}},
			{ID: "final_approval", Name: "最终审批", Description: "CEO审批", Dependencies: []string{"second_level_approval"}},
			{ID: "execute_action", Name: "执行操作", Description: "执行批准的操作", Dependencies: []string{"final_approval"}},
		},
	}
}

// CreateFromTemplate 从模板创建工作流
func (c *Client) CreateFromTemplate(template *WorkflowTemplate, handlers map[string]workflow.TaskHandler) (*EasyWorkflow, error) {
	ew := c.NewEasyWorkflow(template.Name)
	
	// 检查是否所有步骤都有对应的处理器
	for _, step := range template.Steps {
		handler, exists := handlers[step.ID]
		if !exists {
			return nil, fmt.Errorf("缺少步骤 '%s' 的处理器", step.ID)
		}
		ew.Task(step.ID, step.Name, handler)
	}
	
	// 添加依赖关系
	for _, step := range template.Steps {
		for _, dep := range step.Dependencies {
			ew.Then(dep, step.ID)
		}
	}
	
	return ew, nil
}

// BatchExecution 批量执行工具
type BatchExecution struct {
	client     *Client
	workflowID string
	inputs     []map[string]interface{}
	concurrent int
}

// NewBatchExecution 创建批量执行
func (c *Client) NewBatchExecution(workflowID string) *BatchExecution {
	return &BatchExecution{
		client:     c,
		workflowID: workflowID,
		inputs:     make([]map[string]interface{}, 0),
		concurrent: 5, // 默认并发数
	}
}

// AddInput 添加输入数据
func (be *BatchExecution) AddInput(input map[string]interface{}) *BatchExecution {
	be.inputs = append(be.inputs, input)
	return be
}

// SetConcurrency 设置并发数
func (be *BatchExecution) SetConcurrency(concurrent int) *BatchExecution {
	be.concurrent = concurrent
	return be
}

// Execute 执行批量任务
func (be *BatchExecution) Execute(ctx context.Context) ([]*ExecutionResult, error) {
	if len(be.inputs) == 0 {
		return nil, fmt.Errorf("没有输入数据")
	}

	results := make([]*ExecutionResult, len(be.inputs))
	errors := make([]error, len(be.inputs))
	
	// 使用信号量控制并发
	semaphore := make(chan struct{}, be.concurrent)
	done := make(chan struct{})
	
	for i, input := range be.inputs {
		go func(index int, data map[string]interface{}) {
			defer func() { done <- struct{}{} }()
			
			semaphore <- struct{}{} // 获取信号量
			defer func() { <-semaphore }() // 释放信号量
			
			result, err := be.client.Execute(ctx, be.workflowID, data)
			results[index] = result
			errors[index] = err
		}(i, input)
	}
	
	// 等待所有任务完成
	for range be.inputs {
		<-done
	}
	
	// 检查是否有错误
	hasError := false
	for _, err := range errors {
		if err != nil {
			hasError = true
			break
		}
	}
	
	if hasError {
		return results, fmt.Errorf("批量执行中有任务失败")
	}
	
	return results, nil
}

// ScheduledExecution 定时执行工具
type ScheduledExecution struct {
	client     *Client
	workflowID string
	input      map[string]interface{}
	interval   time.Duration
	stopChan   chan struct{}
}

// NewScheduledExecution 创建定时执行
func (c *Client) NewScheduledExecution(workflowID string, input map[string]interface{}, interval time.Duration) *ScheduledExecution {
	return &ScheduledExecution{
		client:     c,
		workflowID: workflowID,
		input:      input,
		interval:   interval,
		stopChan:   make(chan struct{}),
	}
}

// Start 开始定时执行
func (se *ScheduledExecution) Start(ctx context.Context) error {
	ticker := time.NewTicker(se.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-se.stopChan:
			return nil
		case <-ticker.C:
			// 异步执行，不阻塞定时器
			go func() {
				_, err := se.client.ExecuteAsync(ctx, se.workflowID, se.input)
				if err != nil {
					fmt.Printf("定时执行失败: %v\n", err)
				}
			}()
		}
	}
}

// Stop 停止定时执行
func (se *ScheduledExecution) Stop() {
	close(se.stopChan)
}

// ConditionalExecution 条件执行工具
type ConditionalExecution struct {
	client    *Client
	condition func() bool
	trueFlow  string
	falseFlow string
	input     map[string]interface{}
}

// NewConditionalExecution 创建条件执行
func (c *Client) NewConditionalExecution(condition func() bool, trueFlow, falseFlow string, input map[string]interface{}) *ConditionalExecution {
	return &ConditionalExecution{
		client:    c,
		condition: condition,
		trueFlow:  trueFlow,
		falseFlow: falseFlow,
		input:     input,
	}
}

// Execute 执行条件流程
func (ce *ConditionalExecution) Execute(ctx context.Context) (*ExecutionResult, error) {
	var workflowID string
	if ce.condition() {
		workflowID = ce.trueFlow
	} else {
		workflowID = ce.falseFlow
	}
	
	return ce.client.Execute(ctx, workflowID, ce.input)
}

// Monitor 监控工具
type Monitor struct {
	client *Client
}

// NewMonitor 创建监控工具
func (c *Client) NewMonitor() *Monitor {
	return &Monitor{client: c}
}

// WatchExecution 监控执行进度
func (m *Monitor) WatchExecution(executionID string, callback func(*ExecutionStatus)) error {
	for {
		status, err := m.client.GetExecutionStatus(executionID)
		if err != nil {
			return err
		}
		
		callback(status)
		
		if status.Status == "success" || status.Status == "failed" {
			break
		}
		
		time.Sleep(1 * time.Second)
	}
	
	return nil
}

// GetSystemStats 获取系统统计信息
func (m *Monitor) GetSystemStats() (*Statistics, error) {
	workflows, err := m.client.ListWorkflows()
	if err != nil {
		return nil, err
	}
	
	executions, err := m.client.ListExecutions("", 1000, 0) // 获取最近1000次执行
	if err != nil {
		return nil, err
	}
	
	failedExecutions, err := m.client.GetFailedExecutions()
	if err != nil {
		return nil, err
	}
	
	stats := &Statistics{}
	stats.TotalWorkflows = len(workflows)
	
	for _, wf := range workflows {
		if wf.Status == WorkflowStatusPublished {
			stats.PublishedWorkflows++
		}
	}
	
	stats.TotalExecutions = len(executions)
	for _, exec := range executions {
		switch exec.Status {
		case ExecutionStatusSuccess:
			stats.SuccessfulExecutions++
		case ExecutionStatusFailed:
			stats.FailedExecutions++
		case ExecutionStatusRunning:
			stats.RunningExecutions++
		}
	}
	
	stats.PendingRetries = len(failedExecutions)
	
	return stats, nil
}

// ConfigurationHelper 配置助手
type ConfigurationHelper struct{}

// ProductionConfig 生产环境配置
func (ConfigurationHelper) ProductionConfig(dsn string) *ClientConfig {
	return &ClientConfig{
		MySQLDSN:          dsn,
		WebPort:           8080,
		EnableWeb:         true,
		MaxConcurrency:    50,
		DefaultTimeout:    5 * time.Minute,
		MaxAutoRetries:    5,
		RetryDelay:        2 * time.Second,
		BackoffMultiplier: 2.0,
		MaxRetryDelay:     2 * time.Minute,
		LogLevel:          "warn",
	}
}

// DevelopmentConfig 开发环境配置
func (ConfigurationHelper) DevelopmentConfig(dsn string) *ClientConfig {
	return &ClientConfig{
		MySQLDSN:          dsn,
		WebPort:           3000,
		EnableWeb:         true,
		MaxConcurrency:    5,
		DefaultTimeout:    1 * time.Minute,
		MaxAutoRetries:    3,
		RetryDelay:        1 * time.Second,
		BackoffMultiplier: 1.5,
		MaxRetryDelay:     30 * time.Second,
		LogLevel:          "debug",
	}
}

// TestConfig 测试环境配置
func (ConfigurationHelper) TestConfig(dsn string) *ClientConfig {
	return &ClientConfig{
		MySQLDSN:          dsn,
		WebPort:           0, // 随机端口
		EnableWeb:         false,
		MaxConcurrency:    2,
		DefaultTimeout:    10 * time.Second,
		MaxAutoRetries:    1,
		RetryDelay:        100 * time.Millisecond,
		BackoffMultiplier: 1.0,
		MaxRetryDelay:     1 * time.Second,
		LogLevel:          "error",
	}
}
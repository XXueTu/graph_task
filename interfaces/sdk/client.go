package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/XXueTu/graph_task/domain/workflow"
	"github.com/XXueTu/graph_task/engine"
	"github.com/XXueTu/graph_task/interfaces/web"
)

// Client SDK客户端，提供简化的API接口
type Client struct {
	engine      engine.Engine
	webServer   *web.Server
	config      *ClientConfig
	initialized bool
}

// ClientConfig 客户端配置
type ClientConfig struct {
	// 数据库配置
	MySQLDSN string `json:"mysql_dsn"`

	// Web服务配置
	WebPort    int  `json:"web_port"`
	EnableWeb  bool `json:"enable_web"`
	WebEnabled bool `json:"web_enabled"` // 兼容性字段

	// 执行配置
	MaxConcurrency int           `json:"max_concurrency"`
	DefaultTimeout time.Duration `json:"default_timeout"`

	// 重试配置
	MaxAutoRetries    int           `json:"max_auto_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
	MaxRetryDelay     time.Duration `json:"max_retry_delay"`

	// 日志配置
	LogLevel string `json:"log_level"`
	LogFile  string `json:"log_file"`
}

// DefaultClientConfig 默认客户端配置
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		MySQLDSN:          "root:password@tcp(localhost:3306)/graph_task?charset=utf8mb4&parseTime=True&loc=Local",
		WebPort:           8080,
		EnableWeb:         true,
		MaxConcurrency:    10,
		DefaultTimeout:    30 * time.Second,
		MaxAutoRetries:    3,
		RetryDelay:        1 * time.Second,
		BackoffMultiplier: 2.0,
		MaxRetryDelay:     30 * time.Second,
		LogLevel:          "info",
	}
}

// NewClient 创建新的SDK客户端
func NewClient(config *ClientConfig) (*Client, error) {
	if config == nil {
		config = DefaultClientConfig()
	}

	client := &Client{
		config: config,
	}

	if err := client.initialize(); err != nil {
		return nil, fmt.Errorf("初始化客户端失败: %w", err)
	}

	return client, nil
}

// NewSimpleClient 创建简单客户端（使用默认配置）
func NewSimpleClient() (*Client, error) {
	return NewClient(nil)
}

// NewClientWithDSN 使用DSN创建客户端
func NewClientWithDSN(dsn string) (*Client, error) {
	config := DefaultClientConfig()
	config.MySQLDSN = dsn
	return NewClient(config)
}

// initialize 初始化客户端
func (c *Client) initialize() error {
	// 创建引擎配置
	engineConfig := engine.DefaultEngineConfig()
	engineConfig.MySQLDSN = c.config.MySQLDSN

	// 创建引擎
	engine, err := engine.NewEngine(engineConfig)
	if err != nil {
		return fmt.Errorf("创建引擎失败: %w", err)
	}

	c.engine = engine

	// 启动Web服务器（如果启用）
	if c.config.EnableWeb || c.config.WebEnabled {
		if err := c.startWebServer(); err != nil {
			return fmt.Errorf("启动Web服务器失败: %w", err)
		}
	}

	c.initialized = true
	return nil
}

// startWebServer 启动Web服务器
func (c *Client) startWebServer() error {
	// 获取应用服务
	workflowService := c.engine.GetWorkflowService()
	executionService := c.engine.GetExecutionService()
	retryService := c.engine.GetRetryService()

	// 创建Web服务器
	c.webServer = web.NewServer(
		workflowService,
		executionService,
		retryService,
		c.config.WebPort,
	)

	// 在goroutine中启动Web服务器
	go func() {
		if err := c.webServer.Start(); err != nil {
			fmt.Printf("Web服务器启动失败: %v\n", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Close 关闭客户端
func (c *Client) Close() error {
	if c.engine != nil {
		return c.engine.Close()
	}
	return nil
}

// WorkflowBuilder 工作流构建器
type WorkflowBuilder struct {
	client *Client
	engine engine.Engine
}

// CreateWorkflow 创建工作流构建器
func (c *Client) CreateWorkflow(name string) *WorkflowBuilder {
	if !c.initialized {
		panic("客户端未初始化")
	}

	return &WorkflowBuilder{
		client: c,
		engine: c.engine,
	}
}

// SetDescription 设置工作流描述
func (wb *WorkflowBuilder) SetDescription(description string) *WorkflowBuilder {
	// 这里需要实际的实现，暂时返回自身
	return wb
}

// SetVersion 设置工作流版本
func (wb *WorkflowBuilder) SetVersion(version string) *WorkflowBuilder {
	// 这里需要实际的实现，暂时返回自身
	return wb
}

// AddTask 添加任务
func (wb *WorkflowBuilder) AddTask(id, name string, handler workflow.TaskHandler) *WorkflowBuilder {
	// 这里需要实际的实现，暂时返回自身
	return wb
}

// AddDependency 添加依赖关系
func (wb *WorkflowBuilder) AddDependency(from, to string) *WorkflowBuilder {
	// 这里需要实际的实现，暂时返回自身
	return wb
}

// Build 构建工作流
func (wb *WorkflowBuilder) Build() (*workflow.Workflow, error) {
	// 这里需要实际的实现，暂时返回nil
	return nil, fmt.Errorf("WorkflowBuilder.Build() 需要实际实现")
}

// RegisterWorkflow 注册工作流
func (c *Client) RegisterWorkflow(workflow *workflow.Workflow) error {
	if !c.initialized {
		return fmt.Errorf("客户端未初始化")
	}

	return c.engine.PublishWorkflow(workflow)
}

// Execute 同步执行工作流
func (c *Client) Execute(ctx context.Context, workflowID string, input map[string]interface{}) (*ExecutionResult, error) {
	if !c.initialized {
		return nil, fmt.Errorf("客户端未初始化")
	}

	execution, err := c.engine.Execute(ctx, workflowID, input)
	if err != nil {
		return nil, err
	}

	return &ExecutionResult{
		ID:         execution.ID(),
		WorkflowID: execution.WorkflowID(),
		Status:     string(execution.Status()),
		Input:      execution.Input(),
		Output:     execution.Output(),
		StartedAt:  execution.StartTime(),
		EndedAt:    execution.EndTime(),
		Duration:   execution.Duration(),
		RetryCount: execution.RetryCount(),
	}, nil
}

// ExecuteAsync 异步执行工作流
func (c *Client) ExecuteAsync(ctx context.Context, workflowID string, input map[string]interface{}) (string, error) {
	if !c.initialized {
		return "", fmt.Errorf("客户端未初始化")
	}

	return c.engine.ExecuteAsync(ctx, workflowID, input)
}

// GetExecution 获取执行结果
func (c *Client) GetExecution(executionID string) (*ExecutionResult, error) {
	if !c.initialized {
		return nil, fmt.Errorf("客户端未初始化")
	}

	execution, err := c.engine.GetExecution(executionID)
	if err != nil {
		return nil, err
	}

	return &ExecutionResult{
		ID:         execution.ID(),
		WorkflowID: execution.WorkflowID(),
		Status:     string(execution.Status()),
		Input:      execution.Input(),
		Output:     execution.Output(),
		StartedAt:  execution.StartTime(),
		EndedAt:    execution.EndTime(),
		Duration:   execution.Duration(),
		RetryCount: execution.RetryCount(),
	}, nil
}

// ListWorkflows 列出工作流
func (c *Client) ListWorkflows() ([]*WorkflowInfo, error) {
	if !c.initialized {
		return nil, fmt.Errorf("客户端未初始化")
	}

	workflows, err := c.engine.ListWorkflows()
	if err != nil {
		return nil, err
	}

	var result []*WorkflowInfo
	for _, wf := range workflows {
		result = append(result, &WorkflowInfo{
			ID:          wf.ID(),
			Name:        wf.Name(),
			Description: wf.Description(),
			Version:     wf.Version(),
			Status:      string(wf.Status()),
			CreatedAt:   wf.CreatedAt(),
			UpdatedAt:   wf.UpdatedAt(),
			TaskCount:   len(wf.Tasks()),
		})
	}

	return result, nil
}

// ListExecutions 列出执行记录
func (c *Client) ListExecutions(workflowID string, limit, offset int) ([]*ExecutionResult, error) {
	// if !c.initialized {
	// 	return nil, fmt.Errorf("客户端未初始化")
	// }

	// executions, err := c.engine.ListExecutions(workflowID, limit, offset)
	// if err != nil {
	// 	return nil, err
	// }

	var result []*ExecutionResult
	// for _, exec := range executions {
	// 	result = append(result, &ExecutionResult{
	// 		ID:         exec.ID(),
	// 		WorkflowID: exec.WorkflowID(),
	// 		Status:     string(exec.Status()),
	// 		Input:      exec.Input(),
	// 		Output:     exec.Output(),
	// 		StartedAt:  exec.StartedAt(),
	// 		EndedAt:    exec.EndedAt(),
	// 		Duration:   exec.Duration(),
	// 		RetryCount: exec.RetryCount(),
	// 	})
	// }

	return result, nil
}

// GetExecutionStatus 获取执行状态
func (c *Client) GetExecutionStatus(executionID string) (*ExecutionStatus, error) {
	if !c.initialized {
		return nil, fmt.Errorf("客户端未初始化")
	}

	status, err := c.engine.GetExecutionStatus(executionID)
	if err != nil {
		return nil, err
	}

	return &ExecutionStatus{
		ExecutionID:    executionID,
		Status:         string(status.Status()),
		Progress:       status.Progress(),
		CompletedTasks: status.CompletedTasks(),
		TotalTasks:     status.TotalTasks(),
		CurrentTask:    "",
	}, nil
}

// GetExecutionLogs 获取执行日志
func (c *Client) GetExecutionLogs(executionID string, limit, offset int) ([]*LogEntry, error) {
	if !c.initialized {
		return nil, fmt.Errorf("客户端未初始化")
	}

	logs, err := c.engine.GetExecutionLogs(executionID, limit, offset)
	if err != nil {
		return nil, err
	}

	var result []*LogEntry
	for _, log := range logs {
		result = append(result, &LogEntry{
			ID:          log.ID(),
			ExecutionID: log.ExecutionID(),
			TaskID:      log.TaskID(),
			Level:       string(log.Level()),
			Message:     log.Message(),
			Timestamp:   log.Timestamp(),
		})
	}

	return result, nil
}

// ManualRetry 手动重试失败的执行
func (c *Client) ManualRetry(ctx context.Context, executionID string) (*ExecutionResult, error) {
	// if !c.initialized {
	// 	return nil, fmt.Errorf("客户端未初始化")
	// }

	// execution, err := c.engine.ManualRetry(ctx, executionID)
	// if err != nil {
	// 	return nil, err
	// }

	// return &ExecutionResult{
	// 	ID:         execution.ID(),
	// 	WorkflowID: execution.WorkflowID(),
	// 	Status:     string(execution.Status()),
	// 	Input:      execution.Input(),
	// 	Output:     execution.Output(),
	// 	StartedAt:  execution.StartedAt(),
	// 	EndedAt:    execution.EndedAt(),
	// 	Duration:   execution.Duration(),
	// 	RetryCount: execution.RetryCount(),
	// }, nil
	return nil, nil
}

// GetFailedExecutions 获取失败的执行列表
func (c *Client) GetFailedExecutions() ([]*RetryInfo, error) {
	if !c.initialized {
		return nil, fmt.Errorf("客户端未初始化")
	}

	retries, err := c.engine.GetFailedExecutions()
	if err != nil {
		return nil, err
	}

	var result []*RetryInfo
	for _, retry := range retries {
		result = append(result, &RetryInfo{
			ExecutionID:   retry.ExecutionID(),
			WorkflowID:    retry.WorkflowID(),
			FailureReason: retry.FailureReason(),
			RetryCount:    retry.RetryCount(),
			Status:        string(retry.Status()),
			LastRetryAt:   *retry.LastRetryAt(),
		})
	}

	return result, nil
}

// GetWebURL 获取Web管理界面URL
func (c *Client) GetWebURL() string {
	if c.config.EnableWeb || c.config.WebEnabled {
		return fmt.Sprintf("http://localhost:%d", c.config.WebPort)
	}
	return ""
}

// IsWebEnabled 检查Web服务是否启用
func (c *Client) IsWebEnabled() bool {
	return c.config.EnableWeb || c.config.WebEnabled
}

// 便捷函数

// SimpleExecute 简单执行工作流（使用默认上下文）
func (c *Client) SimpleExecute(workflowID string, input map[string]interface{}) (*ExecutionResult, error) {
	ctx := context.Background()
	return c.Execute(ctx, workflowID, input)
}

// SimpleExecuteAsync 简单异步执行工作流
func (c *Client) SimpleExecuteAsync(workflowID string, input map[string]interface{}) (string, error) {
	ctx := context.Background()
	return c.ExecuteAsync(ctx, workflowID, input)
}

// WaitForExecution 等待执行完成
func (c *Client) WaitForExecution(executionID string, timeout time.Duration) (*ExecutionResult, error) {
	start := time.Now()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			result, err := c.GetExecution(executionID)
			if err != nil {
				return nil, err
			}

			if result.Status == "success" || result.Status == "failed" {
				return result, nil
			}

			if time.Since(start) > timeout {
				return result, fmt.Errorf("等待执行超时")
			}
		}
	}
}

// HealthCheck 健康检查
func (c *Client) HealthCheck() error {
	if !c.initialized {
		return fmt.Errorf("客户端未初始化")
	}

	// 简单的健康检查：尝试列出工作流
	_, err := c.ListWorkflows()
	return err
}

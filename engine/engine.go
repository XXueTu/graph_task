package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/XXueTu/graph_task/event"
)

// engine 核心引擎实现
type engine struct {
	storage        Storage
	contextManager *LocalContextManager
	planBuilder    PlanBuilder
	executor       Executor

	eventBus       event.EventBus
	retryManager   *RetryManager // 添加重试管理器
	maxConcurrency int
	retryConfig    *RetryConfig // 添加重试配置
	workflows      map[string]*Workflow
	executions     map[string]*ExecutionResult
	mutex          sync.RWMutex
}

// NewEngine 创建引擎实例
func NewEngine(opts ...EngineOption) Engine {
	e := &engine{
		maxConcurrency: 10,
		workflows:      make(map[string]*Workflow),
		executions:     make(map[string]*ExecutionResult),
		retryConfig:    DefaultRetryConfig(), // 默认重试配置
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(e)
	}

	// 初始化默认组件
	if e.contextManager == nil {
		e.contextManager = NewLocalContextManager(5*time.Minute, 30*time.Minute)
	}
	if e.planBuilder == nil {
		e.planBuilder = NewPlanBuilder(e.maxConcurrency)
	}

	if e.eventBus == nil {
		e.eventBus = event.NewEventBus()
	}

	if e.executor == nil {
		e.executor = NewExecutor(e.maxConcurrency, e.eventBus)
	}

	// 初始化重试管理器
	if e.retryManager == nil {
		// 如果有存储且实现了RetryStorage接口，则使用作为重试存储
		var retryStorage RetryStorage
		if e.storage != nil {
			if rs, ok := e.storage.(RetryStorage); ok {
				retryStorage = rs
			}
		}
		e.retryManager = NewRetryManager(e.retryConfig, retryStorage)
	}
	return e
}

// EngineOption 引擎配置选项
type EngineOption func(*engine)

// WithStorage 设置存储
func WithStorage(storage Storage) EngineOption {
	return func(e *engine) {
		e.storage = storage
	}
}

// WithMaxConcurrency 设置最大并发数
func WithMaxConcurrency(maxConcurrency int) EngineOption {
	return func(e *engine) {
		e.maxConcurrency = maxConcurrency
	}
}

// WithContextManager 设置上下文管理器
func WithContextManager(contextManager *LocalContextManager) EngineOption {
	return func(e *engine) {
		e.contextManager = contextManager
	}
}

// WithRetryConfig 设置重试配置
func WithRetryConfig(config *RetryConfig) EngineOption {
	return func(e *engine) {
		e.retryConfig = config
	}
}

// WithRetryManager 设置重试管理器
func WithRetryManager(retryManager *RetryManager) EngineOption {
	return func(e *engine) {
		e.retryManager = retryManager
	}
}

// CreateWorkflow 创建工作流
func (e *engine) CreateWorkflow(name string) WorkflowBuilder {
	builder := NewWorkflowBuilder()
	builder.SetName(name)
	return builder
}

// GetWorkflow 获取工作流
func (e *engine) GetWorkflow(id string) (*Workflow, error) {
	// 直接从内存获取（不使用缓存，因为处理函数无法序列化）
	e.mutex.RLock()
	if workflow, exists := e.workflows[id]; exists {
		e.mutex.RUnlock()
		return workflow, nil
	}
	e.mutex.RUnlock()
	return nil, fmt.Errorf("workflow not found: %s", id)
}

// PublishWorkflow 发布工作流
func (e *engine) PublishWorkflow(workflow *Workflow) error {
	if workflow == nil {
		return fmt.Errorf("workflow cannot be nil")
	}

	// 验证工作流
	if err := e.planBuilder.Validate(workflow); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}

	// 更新状态
	workflow.Status = WorkflowStatusPublished
	workflow.UpdatedAt = time.Now()

	// 保存到内存
	e.mutex.Lock()
	e.workflows[workflow.ID] = workflow
	e.mutex.Unlock()

	// 不缓存工作流定义（因为处理函数无法序列化）

	// 发布事件
	if e.eventBus != nil {
		event := &event.Event{
			Type:       event.EventWorkflowPublished,
			WorkflowID: workflow.ID,
			Data: map[string]any{
				"name":    workflow.Name,
				"version": workflow.Version,
			},
			Timestamp: time.Now().UnixMilli(),
		}
		e.eventBus.Publish(event)
	}

	return nil
}

// DeleteWorkflow 删除工作流
func (e *engine) DeleteWorkflow(id string) error {
	// 从内存删除
	e.mutex.Lock()
	delete(e.workflows, id)
	e.mutex.Unlock()
	return nil
}

// ListWorkflows 列出工作流
func (e *engine) ListWorkflows() ([]*Workflow, error) {
	// 从内存返回
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	workflows := make([]*Workflow, 0, len(e.workflows))
	for _, workflow := range e.workflows {
		workflows = append(workflows, workflow)
	}

	return workflows, nil
}

// Subscribe 订阅事件
func (e *engine) Subscribe(eventType string, handler event.EventHandler) error {
	return e.eventBus.Subscribe(eventType, handler)
}

// Execute 同步执行工作流（支持自动重试）
func (e *engine) Execute(ctx context.Context, workflowID string, input map[string]any) (*ExecutionResult, error) {
	// 获取工作流
	workflow, err := e.GetWorkflow(workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	if workflow.Status != WorkflowStatusPublished {
		return nil, fmt.Errorf("workflow is not published: %s", workflowID)
	}

	// 构建执行计划
	plan, err := e.buildExecutionPlan(workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to build execution plan: %w", err)
	}

	// 带重试的执行
	result, err := e.executeWithRetry(ctx, plan, workflow, input)

	// 保存执行结果（无论成功失败）
	e.saveExecutionResult(result)

	// 发布事件
	if e.eventBus != nil {
		event := &event.Event{
			Type:        event.EventWorkflowCompleted,
			WorkflowID:  workflowID,
			ExecutionID: result.ExecutionID,
			Data: map[string]any{
				"status":   result.Status,
				"error":    result.Error,
				"input":    result.Input,
				"output":   result.Output,
				"duration": result.Duration.Milliseconds(),
			},
			Timestamp: time.Now().UnixMilli(),
		}
		e.eventBus.Publish(event)
	}

	return result, err
}

// ExecuteAsync 异步执行工作流
func (e *engine) ExecuteAsync(ctx context.Context, workflowID string, input map[string]any) (string, error) {
	// 获取工作流
	workflow, err := e.GetWorkflow(workflowID)
	if err != nil {
		return "", fmt.Errorf("failed to get workflow: %w", err)
	}

	if workflow.Status != WorkflowStatusPublished {
		return "", fmt.Errorf("workflow is not published: %s", workflowID)
	}

	// 构建执行计划
	plan, err := e.buildExecutionPlan(workflow)
	if err != nil {
		return "", fmt.Errorf("failed to build execution plan: %w", err)
	}

	// 异步执行（也支持重试）
	go func() {

		// 带重试的执行
		result, _ := e.executeWithRetry(ctx, plan, workflow, input)

		// 保存执行结果
		e.saveExecutionResult(result)

		// 发布事件
		if e.eventBus != nil {
			event := &event.Event{
				Type:        event.EventWorkflowCompleted,
				WorkflowID:  workflowID,
				ExecutionID: result.ExecutionID,
				Data: map[string]any{
					"status":   result.Status,
					"error":    result.Error,
					"input":    result.Input,
					"output":   result.Output,
					"duration": result.Duration.Milliseconds(),
				},
				Timestamp: time.Now().UnixMilli(),
			}
			e.eventBus.Publish(event)
		}
	}()

	return plan.ExecutionID, nil
}

// GetExecutionResult 获取执行结果
func (e *engine) GetExecutionResult(executionID string) (*ExecutionResult, error) {

	// 从内存获取
	e.mutex.RLock()
	if result, exists := e.executions[executionID]; exists {
		e.mutex.RUnlock()
		return result, nil
	}
	e.mutex.RUnlock()

	// 从存储获取
	if e.storage != nil {
		result, err := e.storage.GetExecution(executionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get execution from storage: %w", err)
		}

		// 更新内存和缓存
		e.mutex.Lock()
		e.executions[executionID] = result
		e.mutex.Unlock()

		return result, nil
	}

	return nil, fmt.Errorf("execution not found: %s", executionID)
}

// CancelExecution 取消执行
func (e *engine) CancelExecution(executionID string) error {
	// 这里需要实现执行取消逻辑
	// 可以通过context取消或者设置取消标志
	return fmt.Errorf("execution cancellation not implemented yet")
}

// GetRunningExecutions 获取正在运行的执行
func (e *engine) GetRunningExecutions() ([]*ExecutionResult, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	var runningExecutions []*ExecutionResult
	for _, execution := range e.executions {
		if execution.Status == WorkflowStatusRunning {
			runningExecutions = append(runningExecutions, execution)
		}
	}

	return runningExecutions, nil
}

// executeWithRetry 带重试的执行工作流
func (e *engine) executeWithRetry(ctx context.Context, plan *ExecutionPlan, workflow *Workflow, input map[string]any) (*ExecutionResult, error) {
	var lastErr error
	var result *ExecutionResult

	for attempt := 0; attempt <= e.retryConfig.MaxAutoRetries; attempt++ {
		// 如果是重试，先等待退避时间
		if attempt > 0 {

			delay := e.calculateRetryDelay(attempt)
			select {
			case <-ctx.Done():

				return e.createFailedResult(plan, workflow, input, ctx.Err()), ctx.Err()
			case <-time.After(delay):
			}
		}

		// 执行工作流
		result, lastErr = e.executor.Execute(ctx, plan, workflow, input, e.contextManager)
		if lastErr == nil {

			result.RetryCount = attempt
			return result, nil
		}

		// 检查是否可重试
		if !e.isRetriableError(lastErr) {
			break
		}
	}

	// 所有重试都失败了，创建失败结果并记录到重试管理器
	failedResult := e.createFailedResult(plan, workflow, input, lastErr)
	failedResult.RetryCount = e.retryConfig.MaxAutoRetries

	// 记录失败执行供手动重试
	e.retryManager.RecordFailedExecution(plan.ExecutionID, workflow.ID, input, lastErr)

	return failedResult, lastErr
}

// createFailedResult 创建失败执行结果
func (e *engine) createFailedResult(plan *ExecutionPlan, workflow *Workflow, input map[string]any, err error) *ExecutionResult {
	now := time.Now()
	return &ExecutionResult{
		ExecutionID: plan.ExecutionID,
		WorkflowID:  workflow.ID,
		Status:      WorkflowStatusFailed,
		Error:       err.Error(),
		Input:       input,
		Output:      make(map[string]interface{}),
		StartTime:   now,
		EndTime:     now,
		Duration:    0,
		TaskResults: make(map[string]*TaskExecutionResult),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// calculateRetryDelay 计算重试延迟
func (e *engine) calculateRetryDelay(attempt int) time.Duration {
	delay := time.Duration(float64(e.retryConfig.RetryDelay.Nanoseconds()) *
		pow(e.retryConfig.BackoffMultiplier, float64(attempt-1)))

	if delay > e.retryConfig.MaxRetryDelay {
		delay = e.retryConfig.MaxRetryDelay
	}

	return delay
}

// isRetriableError 判断错误是否可重试
func (e *engine) isRetriableError(err error) bool {
	// 这里可以根据错误类型判断是否可重试
	// 例如：网络错误可重试，参数错误不可重试
	return true // 简化实现，所有错误都可重试
}

// pow 简单的幂运算
func pow(base, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	result := base
	for i := 1; i < int(exp); i++ {
		result *= base
	}
	return result
}

// buildExecutionPlan 构建执行计划
func (e *engine) buildExecutionPlan(workflow *Workflow) (*ExecutionPlan, error) {
	// 构建新的执行计划
	plan, err := e.planBuilder.Build(workflow)
	if err != nil {
		return nil, err
	}
	return plan, nil
}

// saveExecutionResult 保存执行结果
func (e *engine) saveExecutionResult(result *ExecutionResult) {
	// 保存到内存
	e.mutex.Lock()
	e.executions[result.ExecutionID] = result
	e.mutex.Unlock()
	// 保存到存储
	if e.storage != nil {
		if err := e.storage.SaveExecution(result); err != nil {
			// 记录错误但不影响主流程
			fmt.Printf("Failed to save execution to storage: %v\n", err)
		}

		// 保存任务执行结果
		for _, taskResult := range result.TaskResults {
			if err := e.storage.SaveTaskExecution(result.ExecutionID, taskResult); err != nil {
				fmt.Printf("Failed to save task execution to storage: %v\n", err)
			}
		}
	}
}

// ManualRetry 手动重试失败的执行
func (e *engine) ManualRetry(ctx context.Context, executionID string) (*ExecutionResult, error) {
	retryInfo, err := e.retryManager.GetRetryInfo(executionID)
	if err != nil {
		return nil, fmt.Errorf("retry info not found: %w", err)
	}

	// 重新执行
	result, err := e.Execute(ctx, retryInfo.WorkflowID, retryInfo.Input)

	// 记录手动重试结果
	var newExecutionID string
	var errorMsg string
	success := err == nil

	if result != nil {
		newExecutionID = result.ExecutionID
	}
	if err != nil {
		errorMsg = err.Error()
	}

	if recordErr := e.retryManager.RecordManualRetry(executionID, newExecutionID, success, errorMsg); recordErr != nil {
		fmt.Printf("Failed to record manual retry: %v\n", recordErr)
	}

	return result, err
}

// GetFailedExecutions 获取失败的执行（可手动重试）
func (e *engine) GetFailedExecutions() ([]*RetryInfo, error) {
	return e.retryManager.GetFailedExecutions()
}

// GetRetryStatistics 获取重试统计
func (e *engine) GetRetryStatistics() *RetryStatistics {
	return e.retryManager.GetRetryStatistics()
}

// AbandonRetry 放弃重试失败的执行
func (e *engine) AbandonRetry(executionID string) error {
	return e.retryManager.AbandonRetry(executionID)
}

// Close 关闭引擎
func (e *engine) Close() error {

	// 关闭上下文管理器
	if e.contextManager != nil {
		e.contextManager.Close()
	}

	// 关闭存储
	if e.storage != nil {
		if closer, ok := e.storage.(interface{ Close() error }); ok {
			return closer.Close()
		}
	}
	return nil
}

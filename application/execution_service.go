package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/XXueTu/graph_task/domain/execution"
	"github.com/XXueTu/graph_task/domain/logger"
	"github.com/XXueTu/graph_task/domain/retry"
	"github.com/XXueTu/graph_task/domain/trace"
	"github.com/XXueTu/graph_task/domain/workflow"
)

// ExecutionService 执行应用服务
type ExecutionService struct {
	workflowRepo   workflow.Repository
	executionRepo  execution.Repository
	retryRepo      retry.Repository
	planner        execution.PlannerService
	executor       execution.ExecutorService
	retryConfig    *retry.Config
	monitor        execution.MonitorService
	tracer         execution.TracerService
	logger         logger.LoggerService
	contextManager execution.ContextManagerService
	eventPub       EventPublisher
}

// NewExecutionService 创建执行服务
func NewExecutionService(
	workflowRepo workflow.Repository,
	executionRepo execution.Repository,
	retryRepo retry.Repository,
	planner execution.PlannerService,
	executor execution.ExecutorService,
	retryConfig *retry.Config,
	monitor execution.MonitorService,
	tracer execution.TracerService,
	logger logger.LoggerService,
	contextManager execution.ContextManagerService,
	eventPub EventPublisher,
) *ExecutionService {
	return &ExecutionService{
		workflowRepo:   workflowRepo,
		executionRepo:  executionRepo,
		retryRepo:      retryRepo,
		planner:        planner,
		executor:       executor,
		retryConfig:    retryConfig,
		monitor:        monitor,
		tracer:         tracer,
		logger:         logger,
		contextManager: contextManager,
		eventPub:       eventPub,
	}
}

// Execute 同步执行工作流
func (s *ExecutionService) Execute(ctx context.Context, workflowID string, input map[string]interface{}) (*execution.Execution, error) {
	// 获取工作流
	wf, err := s.workflowRepo.FindByID(workflowID)
	if err != nil {
		return nil, NewApplicationErrorf("workflow not found: %s", workflowID)
	}

	if wf.Status() != workflow.WorkflowStatusPublished {
		return nil, NewApplicationErrorf("workflow is not published: %s", workflowID)
	}

	// 生成执行ID
	executionID := s.generateExecutionID()

	// 构建执行计划
	plan, err := s.planner.BuildPlan(wf, executionID, 10) // 默认并发数10
	if err != nil {
		return nil, fmt.Errorf("failed to build execution plan: %w", err)
	}

	// 带重试的执行
	result, err := s.executeWithRetry(ctx, plan, wf, input)

	// 保存执行结果
	if saveErr := s.executionRepo.SaveExecution(result); saveErr != nil {
		// 记录保存错误但不影响返回结果
		fmt.Printf("Failed to save execution: %v\n", saveErr)
	}

	return result, err
}

// ExecuteAsync 异步执行工作流
func (s *ExecutionService) ExecuteAsync(ctx context.Context, workflowID string, input map[string]interface{}) (string, error) {
	// 获取工作流
	wf, err := s.workflowRepo.FindByID(workflowID)
	if err != nil {
		return "", NewApplicationErrorf("workflow not found: %s", workflowID)
	}

	if wf.Status() != workflow.WorkflowStatusPublished {
		return "", NewApplicationErrorf("workflow is not published: %s", workflowID)
	}

	// 生成执行ID
	executionID := s.generateExecutionID()

	// 构建执行计划
	plan, err := s.planner.BuildPlan(wf, executionID, 10)
	if err != nil {
		return "", fmt.Errorf("failed to build execution plan: %w", err)
	}

	// 异步执行
	go func() {
		result, _ := s.executeWithRetry(ctx, plan, wf, input)
		// 保存执行结果
		if saveErr := s.executionRepo.SaveExecution(result); saveErr != nil {
			fmt.Printf("Failed to save execution: %v\n", saveErr)
		}
	}()

	return executionID, nil
}

// GetExecution 获取执行结果
func (s *ExecutionService) GetExecution(executionID string) (*execution.Execution, error) {
	return s.executionRepo.FindExecutionByID(executionID)
}

// ListExecutions 列出执行记录
func (s *ExecutionService) ListExecutions(workflowID string, limit, offset int) ([]*execution.Execution, error) {
	return s.executionRepo.ListExecutions(workflowID, limit, offset)
}

// CancelExecution 取消执行
func (s *ExecutionService) CancelExecution(executionID string) error {
	exec, err := s.executionRepo.FindExecutionByID(executionID)
	if err != nil {
		return err
	}

	if !exec.IsRunning() {
		return NewApplicationError("execution is not running")
	}

	exec.Cancel()
	return s.executionRepo.UpdateExecution(exec)
}

// GetRunningExecutions 获取正在运行的执行
func (s *ExecutionService) GetRunningExecutions() ([]*execution.Execution, error) {
	return s.executionRepo.ListRunningExecutions()
}

// GetExecutionStatus 获取执行状态
func (s *ExecutionService) GetExecutionStatus(executionID string) (*execution.ExecutionStatus, error) {
	return s.monitor.GetExecutionStatus(executionID)
}

// GetExecutionLogs 获取执行日志
func (s *ExecutionService) GetExecutionLogs(executionID string, limit, offset int) ([]*logger.LogEntry, error) {
	return s.logger.GetLogs(executionID, limit, offset)
}

// GetTaskLogs 获取任务日志
func (s *ExecutionService) GetTaskLogs(executionID, taskID string, limit, offset int) ([]*logger.LogEntry, error) {
	return s.logger.GetTaskLogs(executionID, taskID, limit, offset)
}

// GetExecutionTrace 获取执行追踪
func (s *ExecutionService) GetExecutionTrace(executionID string) (*trace.Trace, error) {
	return s.tracer.GetTrace(executionID)
}

// GetExecutionTaskResults 获取执行的任务结果列表
func (s *ExecutionService) GetExecutionTaskResults(executionID string) ([]*execution.TaskResult, error) {
	exec, err := s.executionRepo.FindExecutionByID(executionID)
	if err != nil {
		return nil, err
	}
	
	taskResults := exec.TaskResults()
	var results []*execution.TaskResult
	for _, taskResult := range taskResults {
		results = append(results, taskResult)
	}
	
	return results, nil
}

// executeWithRetry 带重试的执行
func (s *ExecutionService) executeWithRetry(ctx context.Context, plan *execution.Plan, wf *workflow.Workflow, input map[string]interface{}) (*execution.Execution, error) {
	var lastErr error
	var result *execution.Execution

	for attempt := 0; attempt <= s.retryConfig.MaxAutoRetries(); attempt++ {
		// 如果是重试，先等待退避时间
		if attempt > 0 {
			delay := s.retryConfig.CalculateDelay(attempt)
			select {
			case <-ctx.Done():
				result = s.createFailedExecution(plan.ExecutionID(), plan.WorkflowID(), input, ctx.Err())
				return result, ctx.Err()
			case <-time.After(delay):
				// 等待退避时间
			}
		}

		// 执行工作流
		result, lastErr = s.executor.ExecutePlan(ctx, plan, wf, input)
		if lastErr == nil {
			result.IncrementRetry() // 设置重试次数
			return result, nil
		}

		// 检查是否可重试
		if !s.isRetriableError(lastErr) {
			break
		}
	}

	// 所有重试都失败了，记录到重试管理器
	if result != nil {
		retryInfo := retry.NewInfo(result.ID(), result.WorkflowID(), input, lastErr.Error())
		retryInfo.MarkExhausted()
		if saveErr := s.retryRepo.SaveRetryInfo(retryInfo); saveErr != nil {
			fmt.Printf("Failed to save retry info: %v\n", saveErr)
		}
	}

	return result, lastErr
}

// createFailedExecution 创建失败执行
func (s *ExecutionService) createFailedExecution(executionID, workflowID string, input map[string]interface{}, err error) *execution.Execution {
	exec := execution.NewExecution(executionID, workflowID, input)
	exec.Start()
	exec.Fail(err.Error())
	return exec
}

// isRetriableError 判断错误是否可重试
func (s *ExecutionService) isRetriableError(err error) bool {
	// 这里可以根据错误类型判断是否可重试
	// 例如：网络错误可重试，参数错误不可重试
	return true // 简化实现
}

// generateExecutionID 生成执行ID
func (s *ExecutionService) generateExecutionID() string {
	uuid := uuid.New()
	return fmt.Sprintf("exec_%s", uuid.String())
}

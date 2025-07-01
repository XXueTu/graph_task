package engine

import (
	"context"
	"time"

	"github.com/XXueTu/graph_task/application"
	"github.com/XXueTu/graph_task/domain/execution"
	"github.com/XXueTu/graph_task/domain/logger"
	"github.com/XXueTu/graph_task/domain/retry"
	"github.com/XXueTu/graph_task/domain/trace"
	"github.com/XXueTu/graph_task/domain/workflow"
	"github.com/XXueTu/graph_task/infrastructure/eventbus"
	"github.com/XXueTu/graph_task/infrastructure/persistence/memory"
	"github.com/XXueTu/graph_task/infrastructure/persistence/mysql"
)

// Engine 新的任务编排引擎接口
type Engine interface {
	// 工作流管理
	CreateWorkflow(name string) workflow.Builder
	GetWorkflow(id string) (*workflow.Workflow, error)
	PublishWorkflow(workflow *workflow.Workflow) error
	DeleteWorkflow(id string) error
	ListWorkflows() ([]*workflow.Workflow, error)

	// 执行管理
	Execute(ctx context.Context, workflowID string, input map[string]interface{}) (*execution.Execution, error)
	ExecuteAsync(ctx context.Context, workflowID string, input map[string]interface{}) (string, error)
	GetExecution(executionID string) (*execution.Execution, error)
	CancelExecution(executionID string) error
	GetRunningExecutions() ([]*execution.Execution, error)

	// 执行监控
	GetExecutionStatus(executionID string) (*execution.ExecutionStatus, error)
	GetExecutionLogs(executionID string, limit, offset int) ([]*logger.LogEntry, error)
	GetTaskLogs(executionID, taskID string, limit, offset int) ([]*logger.LogEntry, error)
	GetExecutionTrace(executionID string) (*trace.Trace, error)

	// 重试管理
	ManualRetry(ctx context.Context, executionID string) (*execution.Execution, error)
	GetFailedExecutions() ([]*retry.Info, error)
	GetRetryStatistics() (*retry.Statistics, error)
	AbandonRetry(executionID string) error

	// 事件订阅
	Subscribe(eventType string, handler application.EventHandler) error

	// 获取应用服务
	GetWorkflowService() *application.WorkflowService
	GetExecutionService() *application.ExecutionService
	GetRetryService() *application.RetryService

	// 生命周期管理
	Close() error
}

// EngineConfig 引擎配置
type EngineConfig struct {
	// 数据库配置
	MySQLDSN string

	// 执行配置
	MaxConcurrency int

	// 重试配置
	RetryConfig *retry.Config

	// 批处理配置
	TraceBatchSize     int
	TraceFlushInterval time.Duration
	LogBatchSize       int
	LogFlushInterval   time.Duration

	// 清理配置
	ContextCleanupInterval time.Duration
	MaxContextAge          time.Duration
	StatusCleanupAge       time.Duration
}

// DefaultEngineConfig 默认引擎配置
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		MySQLDSN:               "user:pass@tcp(localhost:3306)/taskdb",
		MaxConcurrency:         10,
		RetryConfig:            retry.DefaultConfig(),
		TraceBatchSize:         100,
		TraceFlushInterval:     5 * time.Second,
		LogBatchSize:           50,
		LogFlushInterval:       2 * time.Second,
		ContextCleanupInterval: 5 * time.Minute,
		MaxContextAge:          30 * time.Minute,
		StatusCleanupAge:       1 * time.Hour,
	}
}

// engine 新的引擎实现
type engine struct {
	// 应用服务
	workflowService  *application.WorkflowService
	executionService *application.ExecutionService
	retryService     *application.RetryService

	// 领域服务
	monitor        execution.MonitorService
	tracer         execution.TracerService
	logger         logger.LoggerService
	contextManager execution.ContextManagerService

	// 基础设施
	eventBus *eventbus.EventBus

	// 配置
	config *EngineConfig
}

// NewEngine 创建新引擎
func NewEngine(config *EngineConfig) (Engine, error) {
	if config == nil {
		config = DefaultEngineConfig()
	}

	// 创建基础设施组件
	eventBus := eventbus.NewEventBus()

	// 创建仓储
	workflowRepo := memory.NewWorkflowRepository()

	executionRepo, err := mysql.NewExecutionRepository(config.MySQLDSN)
	if err != nil {
		return nil, err
	}

	retryRepo, err := mysql.NewRetryRepository(config.MySQLDSN)
	if err != nil {
		return nil, err
	}

	traceRepo, err := mysql.NewTraceRepository(config.MySQLDSN)
	if err != nil {
		return nil, err
	}

	logRepo, err := mysql.NewLogRepository(config.MySQLDSN)
	if err != nil {
		return nil, err
	}

	// 创建领域服务
	planner := execution.NewPlannerService()
	monitor := execution.NewMonitorService()
	tracer := execution.NewTracerService(traceRepo, config.TraceBatchSize, config.TraceFlushInterval)
	logger := logger.NewLoggerService(logRepo, config.LogBatchSize, config.LogFlushInterval)
	contextManager := execution.NewContextManagerService(config.ContextCleanupInterval, config.MaxContextAge)

	// 创建执行器（需要事件发布器适配）
	executor := execution.NewExecutorService(config.MaxConcurrency, &executionEventPublisher{eventBus}, logger)

	// 创建应用服务
	workflowService := application.NewWorkflowService(workflowRepo, eventBus)
	executionService := application.NewExecutionService(
		workflowRepo, executionRepo, retryRepo, planner, executor, config.RetryConfig,
		monitor, tracer, logger, contextManager, eventBus)
	retryService := application.NewRetryService(retryRepo, eventBus)
	retryService.SetExecutionService(executionService)

	return &engine{
		workflowService:  workflowService,
		executionService: executionService,
		retryService:     retryService,
		monitor:          monitor,
		tracer:           tracer,
		logger:           logger,
		contextManager:   contextManager,
		eventBus:         eventBus,
		config:           config,
	}, nil
}

// executionEventPublisher 执行事件发布器适配器
type executionEventPublisher struct {
	eventBus *eventbus.EventBus
}

func (p *executionEventPublisher) Publish(event *execution.Event) error {
	appEvent := application.NewEvent(event.Type(), event.ExecutionID(), map[string]interface{}{
		"workflow_id": event.WorkflowID(),
		"task_id":     event.TaskID(),
		"data":        event.Data(),
		"timestamp":   event.Timestamp(),
	})
	return p.eventBus.Publish(appEvent)
}

// 工作流管理方法实现
func (e *engine) CreateWorkflow(name string) workflow.Builder {
	return e.workflowService.CreateWorkflow(generateWorkflowID(), name)
}

func (e *engine) GetWorkflow(id string) (*workflow.Workflow, error) {
	return e.workflowService.GetWorkflow(id)
}

func (e *engine) PublishWorkflow(wf *workflow.Workflow) error {
	return e.workflowService.PublishWorkflow(wf)
}

func (e *engine) DeleteWorkflow(id string) error {
	return e.workflowService.DeleteWorkflow(id)
}

func (e *engine) ListWorkflows() ([]*workflow.Workflow, error) {
	return e.workflowService.ListWorkflows()
}

// 执行管理方法实现
func (e *engine) Execute(ctx context.Context, workflowID string, input map[string]interface{}) (*execution.Execution, error) {
	return e.executionService.Execute(ctx, workflowID, input)
}

func (e *engine) ExecuteAsync(ctx context.Context, workflowID string, input map[string]interface{}) (string, error) {
	return e.executionService.ExecuteAsync(ctx, workflowID, input)
}

func (e *engine) GetExecution(executionID string) (*execution.Execution, error) {
	return e.executionService.GetExecution(executionID)
}

func (e *engine) CancelExecution(executionID string) error {
	return e.executionService.CancelExecution(executionID)
}

func (e *engine) GetRunningExecutions() ([]*execution.Execution, error) {
	return e.executionService.GetRunningExecutions()
}

// 执行监控方法实现
func (e *engine) GetExecutionStatus(executionID string) (*execution.ExecutionStatus, error) {
	return e.executionService.GetExecutionStatus(executionID)
}

func (e *engine) GetExecutionLogs(executionID string, limit, offset int) ([]*logger.LogEntry, error) {
	return e.executionService.GetExecutionLogs(executionID, limit, offset)
}

func (e *engine) GetTaskLogs(executionID, taskID string, limit, offset int) ([]*logger.LogEntry, error) {
	return e.executionService.GetTaskLogs(executionID, taskID, limit, offset)
}

func (e *engine) GetExecutionTrace(executionID string) (*trace.Trace, error) {
	return e.executionService.GetExecutionTrace(executionID)
}

// 重试管理方法实现
func (e *engine) ManualRetry(ctx context.Context, executionID string) (*execution.Execution, error) {
	record, err := e.retryService.ManualRetry(ctx, executionID)
	if err != nil {
		return nil, err
	}

	if record.Success() && record.ExecutionID() != "" {
		return e.GetExecution(record.ExecutionID())
	}

	return nil, err
}

func (e *engine) GetFailedExecutions() ([]*retry.Info, error) {
	return e.retryService.GetFailedExecutions()
}

func (e *engine) GetRetryStatistics() (*retry.Statistics, error) {
	return e.retryService.GetRetryStatistics()
}

func (e *engine) AbandonRetry(executionID string) error {
	return e.retryService.AbandonRetry(executionID)
}

// 事件订阅方法实现
func (e *engine) Subscribe(eventType string, handler application.EventHandler) error {
	return e.eventBus.Subscribe(eventType, handler)
}

// 获取应用服务
func (e *engine) GetWorkflowService() *application.WorkflowService {
	return e.workflowService
}

func (e *engine) GetExecutionService() *application.ExecutionService {
	return e.executionService
}

func (e *engine) GetRetryService() *application.RetryService {
	return e.retryService
}

// 生命周期管理
func (e *engine) Close() error {
	// 关闭各个组件
	if e.tracer != nil {
		e.tracer.Close()
	}
	if e.logger != nil {
		e.logger.Close()
	}
	if e.contextManager != nil {
		e.contextManager.Close()
	}
	return nil
}

// 工具函数
func generateWorkflowID() string {
	return "workflow_" + time.Now().Format("20060102150405.000000")
}

package engine

import (
	"context"

	"github.com/XXueTu/graph_task/event"
	"github.com/XXueTu/graph_task/types"
)

// Engine 核心引擎接口
type Engine interface {
	// 工作流管理
	CreateWorkflow(name string) WorkflowBuilder
	GetWorkflow(id string) (*Workflow, error)
	PublishWorkflow(workflow *Workflow) error
	DeleteWorkflow(id string) error
	ListWorkflows() ([]*Workflow, error)

	// 事件总线
	Subscribe(eventType string, handler event.EventHandler) error

	// 执行管理
	Execute(ctx context.Context, workflowID string, input map[string]interface{}) (*types.ExecutionResult, error)
	ExecuteAsync(ctx context.Context, workflowID string, input map[string]interface{}) (string, error)
	GetExecutionResult(executionID string) (*types.ExecutionResult, error)
	CancelExecution(executionID string) error

	// 重试管理
	ManualRetry(ctx context.Context, executionID string) (*types.ExecutionResult, error)
	GetFailedExecutions() ([]*types.RetryInfo, error)
	GetRetryStatistics() *RetryStatistics
	AbandonRetry(executionID string) error

	GetRunningExecutions() ([]*types.ExecutionResult, error)

	// 执行追踪
	Close() error
}

// WorkflowBuilder 工作流构建器接口
type WorkflowBuilder interface {
	SetName(name string) WorkflowBuilder
	SetDescription(description string) WorkflowBuilder
	SetVersion(version string) WorkflowBuilder
	AddTask(taskID, name string, handler types.TaskHandler) WorkflowBuilder
	AddDependency(fromTask, toTask string) WorkflowBuilder
	SetTaskTimeout(taskID string, timeout int) WorkflowBuilder
	SetTaskRetry(taskID string, retry int) WorkflowBuilder
	SetTaskInput(taskID string, input map[string]interface{}) WorkflowBuilder
	Build() (*Workflow, error)
}

// Storage 存储接口
type Storage interface {
	// 执行记录存储
	SaveExecution(result *types.ExecutionResult) error
	GetExecution(executionID string) (*types.ExecutionResult, error)
	UpdateExecution(result *types.ExecutionResult) error
	ListExecutions(workflowID string, offset, limit int) ([]*types.ExecutionResult, error)
	// 任务执行记录
	SaveTaskExecution(executionID string, result *types.TaskExecutionResult) error
	GetTaskExecution(executionID, taskID string) (*types.TaskExecutionResult, error)
	ListTaskExecutions(executionID string) ([]*types.TaskExecutionResult, error)
}

// Cache 缓存接口
type Cache interface {
	Set(key string, value interface{}, ttl int) error
	Get(key string) (interface{}, error)
	Delete(key string) error
	Exists(key string) bool
	SetNX(key string, value interface{}, ttl int) (bool, error) // 分布式锁
	Expire(key string, ttl int) error
}

// PlanBuilder 执行计划构建器接口
type PlanBuilder interface {
	Build(workflow *Workflow) (*ExecutionPlan, error)
	Validate(workflow *Workflow) error
}

// Executor 执行器接口
type Executor interface {
	Execute(ctx context.Context, plan *ExecutionPlan, workflow *Workflow, input map[string]any, contextManager *LocalContextManager) (*types.ExecutionResult, error)
	ExecuteTask(ctx context.Context, task *Task, execCtx *ExecutionContext) (*types.TaskExecutionResult, error)
}

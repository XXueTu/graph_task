package engine

import (
	"context"
	"sync"
	"time"

	"github.com/XXueTu/graph_task/types"
)

// Type aliases for compatibility
type TaskStatus = types.TaskStatus
type WorkflowStatus = types.WorkflowStatus
type TaskHandler = types.TaskHandler
type ExecutionResult = types.ExecutionResult
type TaskExecutionResult = types.TaskExecutionResult

// Constants for task status
const (
	TaskStatusPending   = types.TaskStatusPending
	TaskStatusRunning   = types.TaskStatusRunning
	TaskStatusSuccess   = types.TaskStatusSuccess
	TaskStatusFailed    = types.TaskStatusFailed
	TaskStatusSkipped   = types.TaskStatusSkipped
	TaskStatusCanceled  = types.TaskStatusCanceled
)

// Constants for workflow status
const (
	WorkflowStatusDraft     = types.WorkflowStatusDraft
	WorkflowStatusPublished = types.WorkflowStatusPublished
	WorkflowStatusRunning   = types.WorkflowStatusRunning
	WorkflowStatusSuccess   = types.WorkflowStatusSuccess
	WorkflowStatusFailed    = types.WorkflowStatusFailed
	WorkflowStatusCanceled  = types.WorkflowStatusCanceled
)

// Task 任务定义
type Task struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Handler      TaskHandler            `json:"-"`
	Dependencies []string               `json:"dependencies"`
	Timeout      time.Duration          `json:"timeout"`
	Retry        int                    `json:"retry"`
	Input        map[string]interface{} `json:"input"`
	Output       map[string]interface{} `json:"output"`
	Status       TaskStatus             `json:"status"`
	Error        string                 `json:"error,omitempty"`
	StartTime    time.Time              `json:"start_time,omitempty"`
	EndTime      time.Time              `json:"end_time,omitempty"`
	mutex        sync.RWMutex
}

// Workflow 工作流定义
type Workflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Tasks       map[string]*Task       `json:"tasks"`
	StartNodes  []string               `json:"start_nodes"`
	EndNodes    []string               `json:"end_nodes"`
	Status      WorkflowStatus         `json:"status"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	mutex       sync.RWMutex
}

// ExecutionPlan 执行计划
type ExecutionPlan struct {
	WorkflowID     string              `json:"workflow_id"`
	ExecutionID    string              `json:"execution_id"`
	Stages         [][]string          `json:"stages"` // 分层执行计划
	TaskGraph      map[string][]string `json:"task_graph"`
	MaxConcurrency int                 `json:"max_concurrency"`
	CreatedAt      time.Time           `json:"created_at"`
}


// ExecutionContext 执行上下文
type ExecutionContext struct {
	ctx         context.Context
	cancel      context.CancelFunc
	ExecutionID string
	WorkflowID  string
	GlobalData  map[string]any
	TaskResults map[string]*TaskExecutionResult
	createdAt   time.Time // 添加创建时间用于本地清理
	mutex       sync.RWMutex
}

// SetGlobalData 设置全局数据
func (ec *ExecutionContext) SetGlobalData(key string, value interface{}) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	if ec.GlobalData == nil {
		ec.GlobalData = make(map[string]interface{})
	}
	ec.GlobalData[key] = value
}

// GetGlobalData 获取全局数据
func (ec *ExecutionContext) GetGlobalData(key string) (interface{}, bool) {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	if ec.GlobalData == nil {
		return nil, false
	}
	value, exists := ec.GlobalData[key]
	return value, exists
}

// SetTaskResult 设置任务结果
func (ec *ExecutionContext) SetTaskResult(taskID string, result *TaskExecutionResult) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	if ec.TaskResults == nil {
		ec.TaskResults = make(map[string]*TaskExecutionResult)
	}
	ec.TaskResults[taskID] = result
}

// GetTaskResult 获取任务结果
func (ec *ExecutionContext) GetTaskResult(taskID string) (*TaskExecutionResult, bool) {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	if ec.TaskResults == nil {
		return nil, false
	}
	result, exists := ec.TaskResults[taskID]
	return result, exists
}

// Cancel 取消执行
func (ec *ExecutionContext) Cancel() {
	if ec.cancel != nil {
		ec.cancel()
	}
}

// Done 返回取消通道
func (ec *ExecutionContext) Done() <-chan struct{} {
	return ec.ctx.Done()
}

// Err 返回错误
func (ec *ExecutionContext) Err() error {
	return ec.ctx.Err()
}

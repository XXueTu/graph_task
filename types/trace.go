package types

import (
	"context"
	"time"
)

// TraceSpan 追踪跨度数据结构
type TraceSpan struct {
	SpanID       string                 `json:"span_id"`
	TraceID      string                 `json:"trace_id"`
	ParentSpanID *string                `json:"parent_span_id,omitempty"`
	Name         string                 `json:"name"`
	StartTime    int64                  `json:"start_time"`
	EndTime      *int64                 `json:"end_time,omitempty"`
	Duration     *int64                 `json:"duration,omitempty"`
	Status       string                 `json:"status"` // running, success, error
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	Events       []TraceEvent           `json:"events,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// TraceEvent 追踪事件
type TraceEvent struct {
	Timestamp  int64                  `json:"timestamp"`
	Name       string                 `json:"name"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// ExecutionTrace 执行追踪记录
type ExecutionTrace struct {
	TraceID     string `json:"trace_id"`
	WorkflowID  string `json:"workflow_id"`
	ExecutionID string `json:"execution_id"`
	RootSpanID  string `json:"root_span_id"`
	StartTime   int64  `json:"start_time"`
	EndTime     *int64 `json:"end_time,omitempty"`
	Duration    *int64 `json:"duration,omitempty"`
	Status      string `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TaskStatus 任务状态
type TaskStatus int

const (
	TaskStatusPending TaskStatus = iota
	TaskStatusRunning
	TaskStatusSuccess
	TaskStatusFailed
	TaskStatusSkipped
	TaskStatusCanceled
)

// WorkflowStatus 工作流状态
type WorkflowStatus int

const (
	WorkflowStatusDraft WorkflowStatus = iota
	WorkflowStatusPublished
	WorkflowStatusRunning
	WorkflowStatusSuccess
	WorkflowStatusFailed
	WorkflowStatusCanceled
)

// TaskHandler 任务处理函数
type TaskHandler func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)

// ExecutionResult 执行结果
type ExecutionResult struct {
	ExecutionID string                          `json:"execution_id"`
	WorkflowID  string                          `json:"workflow_id"`
	Status      WorkflowStatus                  `json:"status"`
	TaskResults map[string]*TaskExecutionResult `json:"task_results"`
	Input       map[string]interface{}          `json:"input"`
	Output      map[string]interface{}          `json:"output"`
	Error       string                          `json:"error,omitempty"`
	StartTime   time.Time                       `json:"start_time"`
	EndTime     time.Time                       `json:"end_time"`
	Duration    time.Duration                   `json:"duration"`
	RetryCount  int                             `json:"retry_count"`
	CreatedAt   time.Time                       `json:"created_at"`
	UpdatedAt   time.Time                       `json:"updated_at"`
}

// TaskExecutionResult 任务执行结果
type TaskExecutionResult struct {
	TaskID     string                 `json:"task_id"`
	Status     TaskStatus             `json:"status"`
	Input      map[string]interface{} `json:"input"`
	Output     map[string]interface{} `json:"output"`
	Error      string                 `json:"error,omitempty"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Duration   time.Duration          `json:"duration"`
	RetryCount int                    `json:"retry_count"`
}

// RetryInfo 重试信息
type RetryInfo struct {
	ExecutionID   string    `json:"execution_id"`
	WorkflowID    string    `json:"workflow_id"`
	Input         map[string]interface{} `json:"input"`
	FailureReason string    `json:"failure_reason"`
	FailedAt      time.Time `json:"failed_at"`
	RetryCount    int       `json:"retry_count"`
	LastRetryAt   *time.Time `json:"last_retry_at,omitempty"`
	Status        string    `json:"status"` // pending, exhausted, abandoned
	ManualRetries []ManualRetryRecord `json:"manual_retries"`
}

// ManualRetryRecord 手动重试记录
type ManualRetryRecord struct {
	AttemptID    string    `json:"attempt_id"`
	RetryAt      time.Time `json:"retry_at"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
	ExecutionID  string    `json:"execution_id,omitempty"` // 新的执行ID
}
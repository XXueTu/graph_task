package sdk

import "time"

// WorkflowInfo 工作流信息
type WorkflowInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	TaskCount   int       `json:"task_count"`
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	ID         string                 `json:"id"`
	WorkflowID string                 `json:"workflow_id"`
	Status     string                 `json:"status"`
	Input      map[string]interface{} `json:"input"`
	Output     map[string]interface{} `json:"output"`
	StartedAt  time.Time              `json:"started_at"`
	EndedAt    time.Time              `json:"ended_at"`
	Duration   time.Duration          `json:"duration"`
	RetryCount int                    `json:"retry_count"`
}

// IsCompleted 检查执行是否完成
func (er *ExecutionResult) IsCompleted() bool {
	return er.Status == "success" || er.Status == "failed"
}

// IsSuccess 检查执行是否成功
func (er *ExecutionResult) IsSuccess() bool {
	return er.Status == "success"
}

// IsFailed 检查执行是否失败
func (er *ExecutionResult) IsFailed() bool {
	return er.Status == "failed"
}

// IsRunning 检查执行是否在运行中
func (er *ExecutionResult) IsRunning() bool {
	return er.Status == "running"
}

// ExecutionStatus 执行状态
type ExecutionStatus struct {
	ExecutionID    string  `json:"execution_id"`
	Status         string  `json:"status"`
	Progress       float64 `json:"progress"`       // 0-100
	CompletedTasks int     `json:"completed_tasks"`
	TotalTasks     int     `json:"total_tasks"`
	CurrentTask    string  `json:"current_task"`
}

// LogEntry 日志条目
type LogEntry struct {
	ID          string    `json:"id"`
	ExecutionID string    `json:"execution_id"`
	TaskID      string    `json:"task_id"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
}

// RetryInfo 重试信息
type RetryInfo struct {
	ExecutionID   string    `json:"execution_id"`
	WorkflowID    string    `json:"workflow_id"`
	FailureReason string    `json:"failure_reason"`
	RetryCount    int       `json:"retry_count"`
	Status        string    `json:"status"`
	LastRetryAt   time.Time `json:"last_retry_at"`
}

// TaskDefinition 任务定义（用于模板创建）
type TaskDefinition struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Timeout     time.Duration `json:"timeout"`
}

// DependencyDefinition 依赖定义
type DependencyDefinition struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// WorkflowTemplate 工作流模板
type WorkflowTemplate struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Version      string                 `json:"version"`
	Steps        []TemplateStep         `json:"steps"`
	Tasks        []TaskDefinition       `json:"tasks"`
	Dependencies []DependencyDefinition `json:"dependencies"`
}


// ExecutionRequest 执行请求
type ExecutionRequest struct {
	WorkflowID string                 `json:"workflow_id"`
	Input      map[string]interface{} `json:"input"`
	Async      bool                   `json:"async"`
}

// Statistics 统计信息
type Statistics struct {
	TotalWorkflows      int `json:"total_workflows"`
	PublishedWorkflows  int `json:"published_workflows"`
	TotalExecutions     int `json:"total_executions"`
	SuccessfulExecutions int `json:"successful_executions"`
	FailedExecutions    int `json:"failed_executions"`
	RunningExecutions   int `json:"running_executions"`
	PendingRetries      int `json:"pending_retries"`
}

// Health 健康状态
type Health struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
}

// Common constants
const (
	// Workflow statuses
	WorkflowStatusDraft     = "draft"
	WorkflowStatusPublished = "published"

	// Execution statuses
	ExecutionStatusPending  = "pending"
	ExecutionStatusRunning  = "running"
	ExecutionStatusSuccess  = "success"
	ExecutionStatusFailed   = "failed"
	ExecutionStatusCanceled = "canceled"

	// Log levels
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"

	// Retry statuses
	RetryStatusPending   = "pending"
	RetryStatusRetrying  = "retrying"
	RetryStatusExhausted = "exhausted"
	RetryStatusRecovered = "recovered"
)
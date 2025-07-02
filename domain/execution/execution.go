package execution

import (
	"context"
	"sync"
	"time"

	"github.com/XXueTu/graph_task/domain/workflow"
)

// Status 执行状态
type Status int

const (
	StatusPending Status = iota
	StatusRunning
	StatusSuccess
	StatusFailed
	StatusCanceled
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusSuccess:
		return "success"
	case StatusFailed:
		return "failed"
	case StatusCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

// TaskResult 任务执行结果
type TaskResult struct {
	taskID     string
	status     workflow.TaskStatus
	input      map[string]interface{}
	output     map[string]interface{}
	error      string
	startTime  time.Time
	endTime    time.Time
	duration   time.Duration
	retryCount int
	mutex      sync.RWMutex
}

// NewTaskResult 创建任务结果
func NewTaskResult(taskID string) *TaskResult {
	return &TaskResult{
		taskID: taskID,
		status: workflow.TaskStatusPending,
		input:  make(map[string]interface{}),
		output: make(map[string]interface{}),
	}
}

// TaskResult getter methods
func (tr *TaskResult) ID() string                     { return tr.taskID } // Use taskID as ID for now
func (tr *TaskResult) TaskID() string                 { return tr.taskID }
func (tr *TaskResult) Status() workflow.TaskStatus    { return tr.status }
func (tr *TaskResult) Input() map[string]interface{}  { return tr.input }
func (tr *TaskResult) Output() map[string]interface{} { return tr.output }
func (tr *TaskResult) Error() string                  { return tr.error }
func (tr *TaskResult) StartTime() time.Time           { return tr.startTime }
func (tr *TaskResult) EndTime() time.Time             { return tr.endTime }
func (tr *TaskResult) Duration() time.Duration        { return tr.duration }
func (tr *TaskResult) RetryCount() int                { return tr.retryCount }

// Start 开始任务
func (tr *TaskResult) Start(input map[string]interface{}) {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()
	tr.status = workflow.TaskStatusRunning
	tr.input = input
	tr.startTime = time.Now()
}

// Complete 完成任务
func (tr *TaskResult) Complete(output map[string]interface{}) {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()
	tr.status = workflow.TaskStatusSuccess
	tr.output = output
	tr.endTime = time.Now()
	tr.duration = tr.endTime.Sub(tr.startTime)
}

// Fail 任务失败
func (tr *TaskResult) Fail(err string) {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()
	tr.status = workflow.TaskStatusFailed
	tr.error = err
	tr.endTime = time.Now()
	tr.duration = tr.endTime.Sub(tr.startTime)
}

// IncrementRetry 增加重试次数
func (tr *TaskResult) IncrementRetry() {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()
	tr.retryCount++
}

// Plan 执行计划
type Plan struct {
	workflowID     string
	executionID    string
	stages         [][]string
	taskGraph      map[string][]string
	maxConcurrency int
	createdAt      time.Time
}

// NewPlan 创建执行计划
func NewPlan(workflowID, executionID string, maxConcurrency int) *Plan {
	return &Plan{
		workflowID:     workflowID,
		executionID:    executionID,
		stages:         make([][]string, 0),
		taskGraph:      make(map[string][]string),
		maxConcurrency: maxConcurrency,
		createdAt:      time.Now(),
	}
}

// Plan getter methods
func (p *Plan) WorkflowID() string             { return p.workflowID }
func (p *Plan) ExecutionID() string            { return p.executionID }
func (p *Plan) Stages() [][]string             { return p.stages }
func (p *Plan) TaskGraph() map[string][]string { return p.taskGraph }
func (p *Plan) MaxConcurrency() int            { return p.maxConcurrency }
func (p *Plan) CreatedAt() time.Time           { return p.createdAt }

// AddStage 添加执行阶段
func (p *Plan) AddStage(taskIDs []string) {
	p.stages = append(p.stages, taskIDs)
}

// SetTaskGraph 设置任务图
func (p *Plan) SetTaskGraph(taskGraph map[string][]string) {
	p.taskGraph = taskGraph
}

// Context 执行上下文
type Context struct {
	ctx         context.Context
	cancel      context.CancelFunc
	executionID string
	workflowID  string
	globalData  map[string]interface{}
	taskResults map[string]*TaskResult
	createdAt   time.Time
	mutex       sync.RWMutex
}

// NewContext 创建执行上下文
func NewContext(ctx context.Context, executionID, workflowID string, input map[string]interface{}) *Context {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	return &Context{
		ctx:         ctxWithCancel,
		cancel:      cancel,
		executionID: executionID,
		workflowID:  workflowID,
		globalData:  map[string]interface{}{"workflow_input": input},
		taskResults: make(map[string]*TaskResult),
		createdAt:   time.Now(),
	}
}

// Context getter methods
func (ec *Context) ExecutionID() string  { return ec.executionID }
func (ec *Context) WorkflowID() string   { return ec.workflowID }
func (ec *Context) CreatedAt() time.Time { return ec.createdAt }

// SetGlobalData 设置全局数据
func (ec *Context) SetGlobalData(key string, value interface{}) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	ec.globalData[key] = value
}

// GetGlobalData 获取全局数据
func (ec *Context) GetGlobalData(key string) (interface{}, bool) {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	value, exists := ec.globalData[key]
	return value, exists
}

// SetTaskResult 设置任务结果
func (ec *Context) SetTaskResult(taskID string, result *TaskResult) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	ec.taskResults[taskID] = result
}

// GetTaskResult 获取任务结果
func (ec *Context) GetTaskResult(taskID string) (*TaskResult, bool) {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	result, exists := ec.taskResults[taskID]
	return result, exists
}

// GetAllTaskResults 获取所有任务结果
func (ec *Context) GetAllTaskResults() map[string]*TaskResult {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	results := make(map[string]*TaskResult)
	for k, v := range ec.taskResults {
		results[k] = v
	}
	return results
}

// Cancel 取消执行
func (ec *Context) Cancel() {
	if ec.cancel != nil {
		ec.cancel()
	}
}

// Done 返回取消通道
func (ec *Context) Done() <-chan struct{} {
	return ec.ctx.Done()
}

// Err 返回错误
func (ec *Context) Err() error {
	return ec.ctx.Err()
}

// Execution 执行聚合根
type Execution struct {
	id          string
	workflowID  string
	status      Status
	input       map[string]interface{}
	output      map[string]interface{}
	error       string
	startTime   time.Time
	endTime     time.Time
	duration    time.Duration
	retryCount  int
	taskResults map[string]*TaskResult
	createdAt   time.Time
	updatedAt   time.Time
	mutex       sync.RWMutex
}

// NewExecution 创建新执行
func NewExecution(id, workflowID string, input map[string]interface{}) *Execution {
	now := time.Now()
	return &Execution{
		id:          id,
		workflowID:  workflowID,
		status:      StatusPending,
		input:       input,
		output:      make(map[string]interface{}),
		taskResults: make(map[string]*TaskResult),
		createdAt:   now,
		updatedAt:   now,
	}
}

// Execution getter methods
func (e *Execution) ID() string                          { return e.id }
func (e *Execution) WorkflowID() string                  { return e.workflowID }
func (e *Execution) Status() Status                      { return e.status }
func (e *Execution) Input() map[string]interface{}       { return e.input }
func (e *Execution) Output() map[string]interface{}      { return e.output }
func (e *Execution) Error() string                       { return e.error }
func (e *Execution) StartTime() time.Time                { return e.startTime }
func (e *Execution) EndTime() time.Time                  { return e.endTime }
func (e *Execution) Duration() time.Duration             { return e.duration }
func (e *Execution) RetryCount() int                     { return e.retryCount }
func (e *Execution) TaskResults() map[string]*TaskResult { return e.taskResults }
func (e *Execution) CreatedAt() time.Time                { return e.createdAt }
func (e *Execution) UpdatedAt() time.Time                { return e.updatedAt }

// Start 开始执行
func (e *Execution) Start() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.status = StatusRunning
	e.startTime = time.Now()
	e.updatedAt = time.Now()
}

// Complete 完成执行
func (e *Execution) Complete(output map[string]interface{}) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.status = StatusSuccess
	e.output = output
	e.endTime = time.Now()
	e.duration = e.endTime.Sub(e.startTime)
	e.updatedAt = time.Now()
}

// Fail 执行失败
func (e *Execution) Fail(err string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.status = StatusFailed
	e.error = err
	e.endTime = time.Now()
	e.duration = e.endTime.Sub(e.startTime)
	e.updatedAt = time.Now()
}

// Cancel 取消执行
func (e *Execution) Cancel() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.status = StatusCanceled
	e.endTime = time.Now()
	e.duration = e.endTime.Sub(e.startTime)
	e.updatedAt = time.Now()
}

// IncrementRetry 增加重试次数
func (e *Execution) IncrementRetry() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.retryCount++
	e.updatedAt = time.Now()
}

// SetTaskResult 设置任务结果
func (e *Execution) SetTaskResult(taskID string, result *TaskResult) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.taskResults[taskID] = result
	e.updatedAt = time.Now()
}

// GetTaskResult 获取任务结果
func (e *Execution) GetTaskResult(taskID string) (*TaskResult, bool) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	result, exists := e.taskResults[taskID]
	return result, exists
}

// IsRunning 是否正在运行
func (e *Execution) IsRunning() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.status == StatusRunning
}

// IsCompleted 是否已完成
func (e *Execution) IsCompleted() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.status == StatusSuccess || e.status == StatusFailed || e.status == StatusCanceled
}

package workflow

import (
	"sync"
	"time"
)

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

func (ts TaskStatus) String() string {
	switch ts {
	case TaskStatusPending:
		return "pending"
	case TaskStatusRunning:
		return "running"
	case TaskStatusSuccess:
		return "success"
	case TaskStatusFailed:
		return "failed"
	case TaskStatusSkipped:
		return "skipped"
	case TaskStatusCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

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

func (ws WorkflowStatus) String() string {
	switch ws {
	case WorkflowStatusDraft:
		return "draft"
	case WorkflowStatusPublished:
		return "published"
	case WorkflowStatusRunning:
		return "running"
	case WorkflowStatusSuccess:
		return "success"
	case WorkflowStatusFailed:
		return "failed"
	case WorkflowStatusCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

// TaskHandler 任务处理函数
type TaskHandler func(ctx ExecContext, input map[string]interface{}) (map[string]interface{}, error)

// Task 任务实体
type Task struct {
	id           string
	name         string
	handler      TaskHandler
	dependencies []string
	timeout      time.Duration
	retry        int
	input        map[string]interface{}
	output       map[string]interface{}
	status       TaskStatus
	error        string
	startTime    time.Time
	endTime      time.Time
	mutex        sync.RWMutex
}

// NewTask 创建新任务
func NewTask(id, name string, handler TaskHandler) *Task {
	return &Task{
		id:           id,
		name:         name,
		handler:      handler,
		dependencies: make([]string, 0),
		input:        make(map[string]interface{}),
		output:       make(map[string]interface{}),
		status:       TaskStatusPending,
		timeout:      30 * time.Second,
		retry:        0,
	}
}

// Task getter methods
func (t *Task) ID() string                     { return t.id }
func (t *Task) Name() string                   { return t.name }
func (t *Task) Handler() TaskHandler           { return t.handler }
func (t *Task) Dependencies() []string         { return t.dependencies }
func (t *Task) Timeout() time.Duration         { return t.timeout }
func (t *Task) Retry() int                     { return t.retry }
func (t *Task) Input() map[string]interface{}  { return t.input }
func (t *Task) Output() map[string]interface{} { return t.output }
func (t *Task) Status() TaskStatus             { return t.status }
func (t *Task) Error() string                  { return t.error }
func (t *Task) StartTime() time.Time           { return t.startTime }
func (t *Task) EndTime() time.Time             { return t.endTime }

// Task setter methods
func (t *Task) SetTimeout(timeout time.Duration)      { t.timeout = timeout }
func (t *Task) SetRetry(retry int)                    { t.retry = retry }
func (t *Task) SetInput(input map[string]interface{}) { t.input = input }
func (t *Task) AddDependency(taskID string)           { t.dependencies = append(t.dependencies, taskID) }

// SetStatus 设置任务状态
func (t *Task) SetStatus(status TaskStatus) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.status = status
}

// SetError 设置任务错误
func (t *Task) SetError(err string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.error = err
	t.status = TaskStatusFailed
}

// SetOutput 设置任务输出
func (t *Task) SetOutput(output map[string]interface{}) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.output = output
}

// Start 开始任务
func (t *Task) Start() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.status = TaskStatusRunning
	t.startTime = time.Now()
}

// Complete 完成任务
func (t *Task) Complete(output map[string]interface{}) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.status = TaskStatusSuccess
	t.output = output
	t.endTime = time.Now()
}

// Fail 任务失败
func (t *Task) Fail(err string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.status = TaskStatusFailed
	t.error = err
	t.endTime = time.Now()
}

// Workflow 工作流聚合根
type Workflow struct {
	id          string
	name        string
	description string
	version     string
	tasks       map[string]*Task
	startNodes  []string
	endNodes    []string
	status      WorkflowStatus
	input       map[string]interface{}
	output      map[string]interface{}
	createdAt   time.Time
	updatedAt   time.Time
	mutex       sync.RWMutex
}

// NewWorkflow 创建新工作流
func NewWorkflow(id, name string) *Workflow {
	now := time.Now()
	return &Workflow{
		id:        id,
		name:      name,
		tasks:     make(map[string]*Task),
		status:    WorkflowStatusDraft,
		input:     make(map[string]interface{}),
		output:    make(map[string]interface{}),
		createdAt: now,
		updatedAt: now,
	}
}

// Workflow getter methods
func (w *Workflow) ID() string                     { return w.id }
func (w *Workflow) Name() string                   { return w.name }
func (w *Workflow) Description() string            { return w.description }
func (w *Workflow) Version() string                { return w.version }
func (w *Workflow) Tasks() map[string]*Task        { return w.tasks }
func (w *Workflow) StartNodes() []string           { return w.startNodes }
func (w *Workflow) EndNodes() []string             { return w.endNodes }
func (w *Workflow) Status() WorkflowStatus         { return w.status }
func (w *Workflow) Input() map[string]interface{}  { return w.input }
func (w *Workflow) Output() map[string]interface{} { return w.output }
func (w *Workflow) CreatedAt() time.Time           { return w.createdAt }
func (w *Workflow) UpdatedAt() time.Time           { return w.updatedAt }

// Workflow setter methods
func (w *Workflow) SetDescription(description string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.description = description
	w.updatedAt = time.Now()
}

func (w *Workflow) SetVersion(version string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.version = version
	w.updatedAt = time.Now()
}

// AddTask 添加任务
func (w *Workflow) AddTask(task *Task) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if _, exists := w.tasks[task.ID()]; exists {
		return NewWorkflowError("task already exists: " + task.ID())
	}

	w.tasks[task.ID()] = task
	w.updatedAt = time.Now()
	return nil
}

// GetTask 获取任务
func (w *Workflow) GetTask(taskID string) (*Task, error) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	task, exists := w.tasks[taskID]
	if !exists {
		return nil, NewWorkflowError("task not found: " + taskID)
	}
	return task, nil
}

// Publish 发布工作流
func (w *Workflow) Publish() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if err := w.validate(); err != nil {
		return err
	}

	w.status = WorkflowStatusPublished
	w.updatedAt = time.Now()
	return nil
}

// validate 验证工作流
func (w *Workflow) validate() error {
	if len(w.tasks) == 0 {
		return NewWorkflowError("workflow has no tasks")
	}

	// 验证任务依赖
	for _, task := range w.tasks {
		for _, depID := range task.Dependencies() {
			if _, exists := w.tasks[depID]; !exists {
				return NewWorkflowError("dependency not found: " + depID)
			}
		}
	}

	// 检查循环依赖
	if w.hasCyclicDependency() {
		return NewWorkflowError("cyclic dependency detected")
	}

	return nil
}

// hasCyclicDependency 检查循环依赖
func (w *Workflow) hasCyclicDependency() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for taskID := range w.tasks {
		if w.hasCyclicDependencyUtil(taskID, visited, recStack) {
			return true
		}
	}
	return false
}

func (w *Workflow) hasCyclicDependencyUtil(taskID string, visited, recStack map[string]bool) bool {
	visited[taskID] = true
	recStack[taskID] = true

	task := w.tasks[taskID]
	for _, depID := range task.Dependencies() {
		if !visited[depID] {
			if w.hasCyclicDependencyUtil(depID, visited, recStack) {
				return true
			}
		} else if recStack[depID] {
			return true
		}
	}

	recStack[taskID] = false
	return false
}

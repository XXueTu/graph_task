package workflow

import (
	"time"
)

// Builder 工作流构建器
type Builder interface {
	SetName(name string) Builder
	SetDescription(description string) Builder
	SetVersion(version string) Builder
	AddTask(taskID, name string, handler TaskHandler) Builder
	AddDependency(fromTask, toTask string) Builder
	SetTaskTimeout(taskID string, timeout time.Duration) Builder
	SetTaskRetry(taskID string, retry int) Builder
	SetTaskInput(taskID string, input map[string]interface{}) Builder
	Build() (*Workflow, error)
}

// builder 工作流构建器实现
type builder struct {
	workflow *Workflow
}

// NewBuilder 创建工作流构建器
func NewBuilder(id string) Builder {
	return &builder{
		workflow: NewWorkflow(id, ""),
	}
}

func (b *builder) SetName(name string) Builder {
	b.workflow.name = name
	return b
}

func (b *builder) SetDescription(description string) Builder {
	b.workflow.SetDescription(description)
	return b
}

func (b *builder) SetVersion(version string) Builder {
	b.workflow.SetVersion(version)
	return b
}

func (b *builder) AddTask(taskID, name string, handler TaskHandler) Builder {
	task := NewTask(taskID, name, handler)
	b.workflow.AddTask(task)
	return b
}

func (b *builder) AddDependency(fromTask, toTask string) Builder {
	if task, err := b.workflow.GetTask(toTask); err == nil {
		task.AddDependency(fromTask)
	}
	return b
}

func (b *builder) SetTaskTimeout(taskID string, timeout time.Duration) Builder {
	if task, err := b.workflow.GetTask(taskID); err == nil {
		task.SetTimeout(timeout)
	}
	return b
}

func (b *builder) SetTaskRetry(taskID string, retry int) Builder {
	if task, err := b.workflow.GetTask(taskID); err == nil {
		task.SetRetry(retry)
	}
	return b
}

func (b *builder) SetTaskInput(taskID string, input map[string]interface{}) Builder {
	if task, err := b.workflow.GetTask(taskID); err == nil {
		task.SetInput(input)
	}
	return b
}

func (b *builder) Build() (*Workflow, error) {
	if err := b.workflow.validate(); err != nil {
		return nil, err
	}
	return b.workflow, nil
}
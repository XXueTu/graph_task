package engine

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

// workflowBuilder 工作流构建器实现
type workflowBuilder struct {
	workflow *Workflow
	tasks    map[string]*Task
	deps     map[string][]string
}

// NewWorkflowBuilder 创建工作流构建器
func NewWorkflowBuilder() WorkflowBuilder {
	return &workflowBuilder{
		workflow: &Workflow{
			Tasks:      make(map[string]*Task),
			StartNodes: make([]string, 0),
			EndNodes:   make([]string, 0),
			Status:     WorkflowStatusDraft,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		tasks: make(map[string]*Task),
		deps:  make(map[string][]string),
	}
}

// SetName 设置工作流名称
func (wb *workflowBuilder) SetName(name string) WorkflowBuilder {
	wb.workflow.Name = name
	return wb
}

// SetDescription 设置工作流描述
func (wb *workflowBuilder) SetDescription(description string) WorkflowBuilder {
	wb.workflow.Description = description
	return wb
}

// SetVersion 设置工作流版本
func (wb *workflowBuilder) SetVersion(version string) WorkflowBuilder {
	wb.workflow.Version = version
	return wb
}

// AddTask 添加任务
func (wb *workflowBuilder) AddTask(taskID, name string, handler TaskHandler) WorkflowBuilder {
	task := &Task{
		ID:           taskID,
		Name:         name,
		Handler:      handler,
		Dependencies: make([]string, 0),
		Status:       TaskStatusPending,
		Timeout:      30 * time.Second, // 默认30秒超时
		Retry:        0,                // 默认不重试
		Input:        make(map[string]any),
		Output:       make(map[string]any),
	}
	wb.tasks[taskID] = task
	return wb
}

// AddDependency 添加依赖关系
func (wb *workflowBuilder) AddDependency(fromTask, toTask string) WorkflowBuilder {
	if wb.deps[toTask] == nil {
		wb.deps[toTask] = make([]string, 0)
	}
	wb.deps[toTask] = append(wb.deps[toTask], fromTask)
	return wb
}

// SetTaskTimeout 设置任务超时时间
func (wb *workflowBuilder) SetTaskTimeout(taskID string, timeoutSeconds int) WorkflowBuilder {
	if task, exists := wb.tasks[taskID]; exists {
		task.Timeout = time.Duration(timeoutSeconds) * time.Second
	}
	return wb
}

// SetTaskRetry 设置任务重试次数
func (wb *workflowBuilder) SetTaskRetry(taskID string, retry int) WorkflowBuilder {
	if task, exists := wb.tasks[taskID]; exists {
		task.Retry = retry
	}
	return wb
}

// SetTaskInput 设置任务输入
func (wb *workflowBuilder) SetTaskInput(taskID string, input map[string]any) WorkflowBuilder {
	if task, exists := wb.tasks[taskID]; exists {
		task.Input = input
	}
	return wb
}

// Build 构建工作流
func (wb *workflowBuilder) Build() (*Workflow, error) {
	// 验证任务是否为空
	if len(wb.tasks) == 0 {
		return nil, fmt.Errorf("workflow must contain at least one task")
	}

	// 设置任务依赖关系
	for taskID, deps := range wb.deps {
		if task, exists := wb.tasks[taskID]; exists {
			task.Dependencies = deps
		}
	}

	// 复制任务到工作流
	wb.workflow.Tasks = make(map[string]*Task)
	for taskID, task := range wb.tasks {
		wb.workflow.Tasks[taskID] = task
	}

	// 计算开始节点和结束节点
	wb.calculateStartEndNodes()

	// 验证工作流
	if err := wb.validate(); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// 生成工作流ID
	if wb.workflow.ID == "" {
		wb.workflow.ID = generateID()
	}

	wb.workflow.UpdatedAt = time.Now()
	return wb.workflow, nil
}

// calculateStartEndNodes 计算开始节点和结束节点
func (wb *workflowBuilder) calculateStartEndNodes() {
	hasDependencies := make(map[string]bool)
	isDependency := make(map[string]bool)

	// 标记有依赖的任务和被依赖的任务
	for taskID, deps := range wb.deps {
		if len(deps) > 0 {
			hasDependencies[taskID] = true
			for _, dep := range deps {
				isDependency[dep] = true
			}
		}
	}

	// 开始节点：没有依赖的任务
	wb.workflow.StartNodes = make([]string, 0)
	for taskID := range wb.tasks {
		if !hasDependencies[taskID] {
			wb.workflow.StartNodes = append(wb.workflow.StartNodes, taskID)
		}
	}

	// 结束节点：不被任何任务依赖的任务
	wb.workflow.EndNodes = make([]string, 0)
	for taskID := range wb.tasks {
		if !isDependency[taskID] {
			wb.workflow.EndNodes = append(wb.workflow.EndNodes, taskID)
		}
	}
}

// validate 验证工作流
func (wb *workflowBuilder) validate() error {
	// 验证所有依赖的任务都存在
	for taskID, deps := range wb.deps {
		if _, exists := wb.tasks[taskID]; !exists {
			return fmt.Errorf("task %s not found", taskID)
		}
		for _, dep := range deps {
			if _, exists := wb.tasks[dep]; !exists {
				return fmt.Errorf("dependency task %s not found", dep)
			}
		}
	}

	// 验证是否存在循环依赖
	if err := wb.detectCycle(); err != nil {
		return err
	}

	// 验证至少有一个开始节点
	if len(wb.workflow.StartNodes) == 0 {
		return fmt.Errorf("workflow must have at least one start node")
	}

	// 验证至少有一个结束节点
	if len(wb.workflow.EndNodes) == 0 {
		return fmt.Errorf("workflow must have at least one end node")
	}

	return nil
}

// detectCycle 检测循环依赖
func (wb *workflowBuilder) detectCycle() error {
	// 使用DFS检测循环依赖
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for taskID := range wb.tasks {
		if !visited[taskID] {
			if wb.hasCycle(taskID, visited, recStack) {
				return fmt.Errorf("circular dependency detected involving task %s", taskID)
			}
		}
	}
	return nil
}

// hasCycle DFS检测循环依赖
func (wb *workflowBuilder) hasCycle(taskID string, visited, recStack map[string]bool) bool {
	visited[taskID] = true
	recStack[taskID] = true

	// 检查所有依赖此任务的任务
	for dependentTask, deps := range wb.deps {
		for _, dep := range deps {
			if dep == taskID {
				if !visited[dependentTask] {
					if wb.hasCycle(dependentTask, visited, recStack) {
						return true
					}
				} else if recStack[dependentTask] {
					return true
				}
			}
		}
	}

	recStack[taskID] = false
	return false
}

// generateID 生成唯一ID
func generateID() string {
	return fmt.Sprintf("wf_%s", time.Now().Format("20060102150405")+"0"+strconv.Itoa(rand.Intn(1000)))
}

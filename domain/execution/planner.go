package execution

import (
	"github.com/XXueTu/graph_task/domain/workflow"
)

// PlannerService 执行计划构建器领域服务
type PlannerService interface {
	// BuildPlan 构建执行计划
	BuildPlan(wf *workflow.Workflow, executionID string, maxConcurrency int) (*Plan, error)
	
	// ValidateWorkflow 验证工作流
	ValidateWorkflow(wf *workflow.Workflow) error
}

// plannerService 执行计划构建器实现
type plannerService struct{}

// NewPlannerService 创建执行计划构建器
func NewPlannerService() PlannerService {
	return &plannerService{}
}

// BuildPlan 构建执行计划
func (p *plannerService) BuildPlan(wf *workflow.Workflow, executionID string, maxConcurrency int) (*Plan, error) {
	if err := p.ValidateWorkflow(wf); err != nil {
		return nil, err
	}

	plan := NewPlan(wf.ID(), executionID, maxConcurrency)
	
	// 构建任务图
	taskGraph := make(map[string][]string)
	for _, task := range wf.Tasks() {
		taskGraph[task.ID()] = task.Dependencies()
	}
	plan.SetTaskGraph(taskGraph)
	
	// 分层构建执行阶段
	stages := p.buildStages(wf.Tasks())
	for _, stage := range stages {
		plan.AddStage(stage)
	}
	
	return plan, nil
}

// ValidateWorkflow 验证工作流
func (p *plannerService) ValidateWorkflow(wf *workflow.Workflow) error {
	if len(wf.Tasks()) == 0 {
		return NewExecutionError("workflow has no tasks")
	}
	
	// 验证任务依赖
	for _, task := range wf.Tasks() {
		for _, depID := range task.Dependencies() {
			if _, err := wf.GetTask(depID); err != nil {
				return NewExecutionErrorf("dependency not found: %s", depID)
			}
		}
	}
	
	// 检查循环依赖
	if p.hasCyclicDependency(wf.Tasks()) {
		return NewExecutionError("cyclic dependency detected")
	}
	
	return nil
}

// buildStages 构建执行阶段
func (p *plannerService) buildStages(tasks map[string]*workflow.Task) [][]string {
	var stages [][]string
	processed := make(map[string]bool)
	
	for len(processed) < len(tasks) {
		var currentStage []string
		
		// 找到所有依赖已满足的任务
		for taskID, task := range tasks {
			if processed[taskID] {
				continue
			}
			
			canExecute := true
			for _, depID := range task.Dependencies() {
				if !processed[depID] {
					canExecute = false
					break
				}
			}
			
			if canExecute {
				currentStage = append(currentStage, taskID)
			}
		}
		
		// 如果没有可执行的任务，说明有循环依赖
		if len(currentStage) == 0 {
			break
		}
		
		// 标记为已处理
		for _, taskID := range currentStage {
			processed[taskID] = true
		}
		
		stages = append(stages, currentStage)
	}
	
	return stages
}

// hasCyclicDependency 检查循环依赖
func (p *plannerService) hasCyclicDependency(tasks map[string]*workflow.Task) bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	
	for taskID := range tasks {
		if p.hasCyclicDependencyUtil(taskID, tasks, visited, recStack) {
			return true
		}
	}
	return false
}

func (p *plannerService) hasCyclicDependencyUtil(taskID string, tasks map[string]*workflow.Task, visited, recStack map[string]bool) bool {
	visited[taskID] = true
	recStack[taskID] = true
	
	task := tasks[taskID]
	for _, depID := range task.Dependencies() {
		if !visited[depID] {
			if p.hasCyclicDependencyUtil(depID, tasks, visited, recStack) {
				return true
			}
		} else if recStack[depID] {
			return true
		}
	}
	
	recStack[taskID] = false
	return false
}
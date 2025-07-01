package engine

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

// planBuilder 执行计划构建器实现
type planBuilder struct {
	maxConcurrency int
}

// NewPlanBuilder 创建计划构建器
func NewPlanBuilder(maxConcurrency int) PlanBuilder {
	if maxConcurrency <= 0 {
		maxConcurrency = 10 // 默认最大并发数
	}
	return &planBuilder{
		maxConcurrency: maxConcurrency,
	}
}

// Build 构建执行计划
func (pb *planBuilder) Build(workflow *Workflow) (*ExecutionPlan, error) {
	if err := pb.Validate(workflow); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// 构建任务依赖图
	taskGraph := pb.buildTaskGraph(workflow)

	// 生成分层执行计划
	stages := pb.generateStages(workflow, taskGraph)

	plan := &ExecutionPlan{
		WorkflowID:     workflow.ID,
		ExecutionID:    generateExecutionID(),
		Stages:         stages,
		TaskGraph:      taskGraph,
		MaxConcurrency: pb.maxConcurrency,
		CreatedAt:      time.Now(),
	}

	return plan, nil
}

// Validate 验证工作流
func (pb *planBuilder) Validate(workflow *Workflow) error {
	if workflow == nil {
		return fmt.Errorf("workflow cannot be nil")
	}

	if len(workflow.Tasks) == 0 {
		return fmt.Errorf("workflow must contain at least one task")
	}

	// 验证所有任务都有处理函数
	for taskID, task := range workflow.Tasks {
		if task.Handler == nil {
			return fmt.Errorf("task %s must have a handler", taskID)
		}
	}

	// 验证依赖关系
	for taskID, task := range workflow.Tasks {
		for _, dep := range task.Dependencies {
			if _, exists := workflow.Tasks[dep]; !exists {
				return fmt.Errorf("task %s depends on non-existent task %s", taskID, dep)
			}
		}
	}

	// 验证循环依赖
	if err := pb.detectCircularDependency(workflow); err != nil {
		return err
	}

	return nil
}

// buildTaskGraph 构建任务依赖图
func (pb *planBuilder) buildTaskGraph(workflow *Workflow) map[string][]string {
	taskGraph := make(map[string][]string)

	for taskID, task := range workflow.Tasks {
		taskGraph[taskID] = make([]string, len(task.Dependencies))
		copy(taskGraph[taskID], task.Dependencies)
	}

	return taskGraph
}

// generateStages 生成分层执行计划
func (pb *planBuilder) generateStages(workflow *Workflow, taskGraph map[string][]string) [][]string {
	stages := make([][]string, 0)
	processed := make(map[string]bool)

	for len(processed) < len(workflow.Tasks) {
		currentStage := make([]string, 0)

		// 找到当前可以执行的任务（所有依赖都已完成）
		for taskID := range workflow.Tasks {
			if processed[taskID] {
				continue
			}

			canExecute := true
			for _, dep := range taskGraph[taskID] {
				if !processed[dep] {
					canExecute = false
					break
				}
			}

			if canExecute {
				currentStage = append(currentStage, taskID)
			}
		}

		// 如果没有可执行的任务，说明存在循环依赖
		if len(currentStage) == 0 {
			break
		}

		// 标记当前阶段的任务为已处理
		for _, taskID := range currentStage {
			processed[taskID] = true
		}

		stages = append(stages, currentStage)
	}

	return stages
}

// detectCircularDependency 检测循环依赖
func (pb *planBuilder) detectCircularDependency(workflow *Workflow) error {
	// 使用拓扑排序检测循环依赖
	inDegree := make(map[string]int)

	// 初始化入度
	for taskID := range workflow.Tasks {
		inDegree[taskID] = 0
	}

	// 计算入度
	for _, task := range workflow.Tasks {
		for range task.Dependencies {
			inDegree[task.ID]++
		}
	}

	// 队列存储入度为0的节点
	queue := make([]string, 0)
	for taskID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, taskID)
		}
	}

	processed := 0

	// 拓扑排序
	for len(queue) > 0 {
		// 取出队首元素
		current := queue[0]
		queue = queue[1:]
		processed++

		// 遍历所有依赖当前任务的任务
		for taskID, task := range workflow.Tasks {
			for _, dep := range task.Dependencies {
				if dep == current {
					inDegree[taskID]--
					if inDegree[taskID] == 0 {
						queue = append(queue, taskID)
					}
				}
			}
		}
	}

	// 如果处理的任务数量不等于总任务数，说明存在循环依赖
	if processed != len(workflow.Tasks) {
		return fmt.Errorf("circular dependency detected in workflow")
	}

	return nil
}

// OptimizePlan 优化执行计划
func (pb *planBuilder) OptimizePlan(plan *ExecutionPlan, workflow *Workflow) *ExecutionPlan {
	// 优化并发度：合并可以并行执行的阶段
	optimizedStages := make([][]string, 0)

	for _, stage := range plan.Stages {
		if len(optimizedStages) == 0 {
			optimizedStages = append(optimizedStages, stage)
			continue
		}

		// 检查当前阶段是否可以与上一个阶段合并
		lastStage := optimizedStages[len(optimizedStages)-1]
		canMerge := len(lastStage)+len(stage) <= pb.maxConcurrency

		if canMerge {
			// 检查依赖关系是否允许合并
			canMergeByDeps := true
			for _, taskID := range stage {
				task := workflow.Tasks[taskID]
				for _, dep := range task.Dependencies {
					for _, lastStageTask := range lastStage {
						if dep == lastStageTask {
							canMergeByDeps = false
							break
						}
					}
					if !canMergeByDeps {
						break
					}
				}
				if !canMergeByDeps {
					break
				}
			}

			if canMergeByDeps {
				// 合并阶段
				optimizedStages[len(optimizedStages)-1] = append(lastStage, stage...)
			} else {
				optimizedStages = append(optimizedStages, stage)
			}
		} else {
			optimizedStages = append(optimizedStages, stage)
		}
	}

	plan.Stages = optimizedStages
	return plan
}

// generateExecutionID 生成执行ID
func generateExecutionID() string {
	return fmt.Sprintf("exec_%s", time.Now().Format("20060102150405")+"0"+strconv.Itoa(rand.Intn(1000)))
}

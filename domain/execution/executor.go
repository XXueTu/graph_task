package execution

import (
	"context"
	"sync"

	"github.com/XXueTu/graph_task/domain/logger"
	"github.com/XXueTu/graph_task/domain/workflow"
)

// ExecutorService 执行器领域服务
type ExecutorService interface {
	// ExecutePlan 执行计划
	ExecutePlan(ctx context.Context, plan *Plan, wf *workflow.Workflow, input map[string]interface{}) (*Execution, error)

	// ExecuteTask 执行单个任务
	ExecuteTask(ctx context.Context, task *workflow.Task, execCtx *Context) (*TaskResult, error)
}

// executorService 执行器实现
type executorService struct {
	maxConcurrency int
	eventPublisher EventPublisher
	logger         logger.LoggerService
}

// NewExecutorService 创建执行器
func NewExecutorService(maxConcurrency int, eventPublisher EventPublisher, logger logger.LoggerService) ExecutorService {
	return &executorService{
		maxConcurrency: maxConcurrency,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// ExecutePlan 执行计划
func (e *executorService) ExecutePlan(ctx context.Context, plan *Plan, wf *workflow.Workflow, input map[string]interface{}) (*Execution, error) {
	// 创建执行记录
	execution := NewExecution(plan.ExecutionID(), plan.WorkflowID(), input)
	execution.Start()

	// 发布执行开始事件
	e.publishEvent("execution.started", execution.ID(), execution.WorkflowID(), "", map[string]interface{}{
		"input": input,
	})

	// 创建执行上下文
	execCtx := NewContext(ctx, execution.ID(), execution.WorkflowID(), input)
	defer execCtx.Cancel()

	// 按阶段执行任务
	for stageIndex, stage := range plan.Stages() {
		if err := e.executeStage(ctx, stage, wf, execCtx, stageIndex); err != nil {
			execution.Fail(err.Error())
			e.publishEvent("execution.failed", execution.ID(), execution.WorkflowID(), "", map[string]interface{}{
				"error": err.Error(),
			})
			return execution, err
		}

		// 检查上下文是否被取消
		if execCtx.Err() != nil {
			execution.Cancel()
			e.publishEvent("execution.cancelled", execution.ID(), execution.WorkflowID(), "", map[string]interface{}{})
			return execution, execCtx.Err()
		}
	}

	// 设置执行结果
	taskResults := execCtx.GetAllTaskResults()
	for taskID, result := range taskResults {
		execution.SetTaskResult(taskID, result)
	}

	// 设置执行结果为空
	execution.Complete(map[string]interface{}{})
	e.publishEvent("execution.completed", execution.ID(), execution.WorkflowID(), "", map[string]interface{}{})

	return execution, nil
}

// executeStage 执行阶段
func (e *executorService) executeStage(ctx context.Context, stage []string, wf *workflow.Workflow, execCtx *Context, stageIndex int) error {
	// 发布阶段开始事件
	e.publishEvent("stage.started", execCtx.ExecutionID(), execCtx.WorkflowID(), "", map[string]interface{}{
		"stage_index": stageIndex,
		"tasks":       stage,
	})

	// 控制并发数
	semaphore := make(chan struct{}, e.maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var stageErr error

	for _, taskID := range stage {
		task, err := wf.GetTask(taskID)
		if err != nil {
			return err
		}

		wg.Add(1)
		go func(t *workflow.Task) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 执行任务
			result, err := e.ExecuteTask(ctx, t, execCtx)

			mu.Lock()
			if err != nil && stageErr == nil {
				stageErr = err
			}
			if result != nil {
				execCtx.SetTaskResult(t.ID(), result)
			}
			mu.Unlock()
		}(task)
	}

	wg.Wait()

	if stageErr != nil {
		e.publishEvent("stage.failed", execCtx.ExecutionID(), execCtx.WorkflowID(), "", map[string]interface{}{
			"stage_index": stageIndex,
			"error":       stageErr.Error(),
		})
		return stageErr
	}

	e.publishEvent("stage.completed", execCtx.ExecutionID(), execCtx.WorkflowID(), "", map[string]interface{}{
		"stage_index": stageIndex,
	})

	return nil
}

// ExecuteTask 执行单个任务
func (e *executorService) ExecuteTask(ctx context.Context, task *workflow.Task, execCtx *Context) (*TaskResult, error) {
	result := NewTaskResult(task.ID())

	// 准备任务输入
	taskInput := e.prepareTaskInput(task, execCtx)
	result.Start(taskInput)

	// 发布任务开始事件
	e.publishEvent("task.started", execCtx.ExecutionID(), execCtx.WorkflowID(), task.ID(), map[string]interface{}{
		"input": taskInput,
	})

	// 创建任务上下文（带超时）
	taskCtx, cancel := context.WithTimeout(ctx, task.Timeout())
	defer cancel()
	svcCtx := workflow.NewExecContext(taskCtx, execCtx.executionID, task.ID(), e.logger)

	// 执行任务
	output, err := task.Handler()(svcCtx, taskInput)
	if err != nil {
		result.Fail(err.Error())
		e.publishEvent("task.failed", execCtx.ExecutionID(), execCtx.WorkflowID(), task.ID(), map[string]interface{}{
			"error": err.Error(),
		})
		return result, err
	}

	result.Complete(output)
	e.publishEvent("task.completed", execCtx.ExecutionID(), execCtx.WorkflowID(), task.ID(), map[string]interface{}{
		"output": output,
	})

	return result, nil
}

// prepareTaskInput 准备任务输入
func (e *executorService) prepareTaskInput(task *workflow.Task, execCtx *Context) map[string]interface{} {
	input := make(map[string]interface{})

	// 复制任务本身的输入
	for k, v := range task.Input() {
		input[k] = v
	}

	// 添加依赖任务的输出
	for _, depID := range task.Dependencies() {
		if depResult, exists := execCtx.GetTaskResult(depID); exists {
			input[depID+"_output"] = depResult.Output()
		}
	}

	// 添加全局数据
	if globalData, exists := execCtx.GetGlobalData("workflow_input"); exists {
		input["workflow_input"] = globalData
	}

	return input
}

// publishEvent 发布事件
func (e *executorService) publishEvent(eventType, executionID, workflowID, taskID string, data map[string]interface{}) {
	if e.eventPublisher != nil {
		event := NewExecutionEvent(eventType, executionID, workflowID, taskID, data)
		e.eventPublisher.Publish(event)
	}
}

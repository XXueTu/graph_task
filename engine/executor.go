package engine

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/XXueTu/graph_task/event"
)

// executor 执行器实现
type executor struct {
	maxConcurrency int
	workerPool     chan struct{}
	eventBus       event.EventBus
}

// NewExecutor 创建执行器
func NewExecutor(maxConcurrency int, eventBus event.EventBus) Executor {
	if maxConcurrency <= 0 {
		maxConcurrency = runtime.NumCPU() * 2
	}
	return &executor{
		maxConcurrency: maxConcurrency,
		workerPool:     make(chan struct{}, maxConcurrency),
		eventBus:       eventBus,
	}
}

// Execute 执行工作流
func (e *executor) Execute(ctx context.Context, plan *ExecutionPlan, workflow *Workflow, input map[string]any, contextManager *LocalContextManager) (*ExecutionResult, error) {
	// 创建执行上下文
	var execCtx *ExecutionContext
	if contextManager != nil {
		execCtx = contextManager.CreateContext(plan.ExecutionID, plan.WorkflowID, input)
		execCtx.ctx, execCtx.cancel = context.WithCancel(ctx)
	} else {
		// 兼容原有方式
		execCtx = &ExecutionContext{
			ExecutionID: plan.ExecutionID,
			WorkflowID:  plan.WorkflowID,
			GlobalData:  input,
			TaskResults: make(map[string]*TaskExecutionResult),
		}
		execCtx.ctx, execCtx.cancel = context.WithCancel(ctx)
	}

	// 创建执行结果
	result := &ExecutionResult{
		ExecutionID: plan.ExecutionID,
		WorkflowID:  plan.WorkflowID,
		Status:      WorkflowStatusRunning,
		TaskResults: make(map[string]*TaskExecutionResult),
		Input:       input,
		Output:      make(map[string]any),
		StartTime:   time.Now(),
	}

	// 并发执行各个阶段
	for stageIndex, stage := range plan.Stages {
		select {
		case <-execCtx.Done():
			result.Status = WorkflowStatusCanceled
			result.Error = "execution canceled"
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, execCtx.Err()
		default:
		}

		// 并发执行当前阶段的所有任务
		if err := e.executeStage(execCtx, stage, workflow, result); err != nil {
			result.Status = WorkflowStatusFailed
			result.Error = fmt.Sprintf("stage %d execution failed: %v", stageIndex, err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, err
		}
	}

	// 设置工作流输出
	result.Output = e.collectWorkflowOutput(workflow, result.TaskResults)
	result.Status = WorkflowStatusSuccess
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// executeStage 执行阶段
func (e *executor) executeStage(execCtx *ExecutionContext, stage []string, workflow *Workflow, result *ExecutionResult) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(stage))

	// 并发执行阶段内的所有任务
	for _, taskID := range stage {
		task, exists := workflow.Tasks[taskID]
		if !exists {
			return fmt.Errorf("task %s not found in workflow", taskID)
		}

		wg.Add(1)
		go func(t *Task) {
			defer wg.Done()

			// 获取工作池令牌
			e.workerPool <- struct{}{}
			defer func() { <-e.workerPool }()

			// 执行任务
			var taskResult *TaskExecutionResult
			var err error
			taskResult, err = e.ExecuteTask(execCtx.ctx, t, execCtx)
			// 发布事件
			if e.eventBus != nil {
				e.eventBus.Publish(&event.Event{
					Type:        event.EventTracePublished,
					WorkflowID:  execCtx.WorkflowID,
					ExecutionID: execCtx.ExecutionID,
					TaskID:      t.ID,
					Timestamp:   time.Now().UnixMilli(),
					Data: map[string]any{
						"task_id":     t.ID,
						"input":       taskResult.Input,
						"output":      taskResult.Output,
						"duration":    taskResult.Duration.Milliseconds(),
						"error":       taskResult.Error,
						"status":      taskResult.Status,
						"retry_count": taskResult.RetryCount,
						"start_time":  taskResult.StartTime.UnixMilli(),
						"end_time":    taskResult.EndTime.UnixMilli(),
					},
				})
			}
			if err != nil {
				errChan <- fmt.Errorf("task %s failed: %w", t.ID, err)
				return
			}

			// 保存任务结果
			execCtx.SetTaskResult(t.ID, taskResult)
			result.TaskResults[t.ID] = taskResult

		}(task)
	}

	// 等待所有任务完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("stage execution failed with %d errors: %v", len(errors), errors[0])
	}

	return nil
}

// ExecuteTask 执行单个任务
func (e *executor) ExecuteTask(ctx context.Context, task *Task, execCtx *ExecutionContext) (*TaskExecutionResult, error) {
	result := &TaskExecutionResult{
		TaskID:    task.ID,
		Status:    TaskStatusRunning,
		Input:     make(map[string]any),
		Output:    make(map[string]any),
		StartTime: time.Now(),
	}

	// 设置任务超时
	taskCtx := ctx
	if task.Timeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(ctx, task.Timeout)
		defer cancel()
	}

	// 准备任务输入
	taskInput := e.prepareTaskInput(task, execCtx)
	result.Input = taskInput

	// 执行任务，支持重试
	var err error
	var output map[string]any

	for attempt := 0; attempt <= task.Retry; attempt++ {
		result.RetryCount = attempt

		select {
		case <-taskCtx.Done():
			result.Status = TaskStatusCanceled
			result.Error = taskCtx.Err().Error()
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, taskCtx.Err()
		default:
		}

		// 执行任务处理函数
		output, err = task.Handler(taskCtx, taskInput)
		if err == nil {
			// 执行成功
			result.Status = TaskStatusSuccess
			result.Output = output
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)

			// 更新任务状态
			task.mutex.Lock()
			task.Status = TaskStatusSuccess
			task.Output = output
			task.EndTime = time.Now()
			task.mutex.Unlock()

			return result, nil
		}

		// 如果是最后一次尝试，或者是不可重试的错误，则失败
		if attempt == task.Retry {
			break
		}

		// 重试前等待一段时间
		select {
		case <-taskCtx.Done():
			result.Status = TaskStatusCanceled
			result.Error = taskCtx.Err().Error()
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)

			return result, taskCtx.Err()
		case <-time.After(time.Duration(attempt+1) * time.Second):
			// 指数退避重试
		}
	}

	// 执行失败
	result.Status = TaskStatusFailed
	result.Error = err.Error()
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// 更新任务状态
	task.mutex.Lock()
	task.Status = TaskStatusFailed
	task.Error = err.Error()
	task.EndTime = time.Now()
	task.mutex.Unlock()

	return result, err
}

// prepareTaskInput 准备任务输入
func (e *executor) prepareTaskInput(task *Task, execCtx *ExecutionContext) map[string]any {
	input := make(map[string]any)

	// 复制任务定义的输入
	for k, v := range task.Input {
		input[k] = v
	}

	// 添加全局数据
	for k, v := range execCtx.GlobalData {
		if _, exists := input[k]; !exists {
			input[k] = v
		}
	}

	// 添加依赖任务的输出
	for _, depTaskID := range task.Dependencies {
		if depResult, exists := execCtx.GetTaskResult(depTaskID); exists {
			// 将依赖任务的输出添加到当前任务的输入中
			for k, v := range depResult.Output {
				inputKey := fmt.Sprintf("%s_%s", depTaskID, k)
				input[inputKey] = v
			}
			// 同时也可以直接访问依赖任务的整个输出
			input[depTaskID] = depResult.Output
		}
	}

	return input
}

// collectWorkflowOutput 收集工作流输出
func (e *executor) collectWorkflowOutput(workflow *Workflow, taskResults map[string]*TaskExecutionResult) map[string]any {
	output := make(map[string]any)

	// 收集所有结束节点的输出
	for _, endNodeID := range workflow.EndNodes {
		if result, exists := taskResults[endNodeID]; exists {
			for k, v := range result.Output {
				output[k] = v
			}
			// 同时保存任务级别的输出
			output[endNodeID] = result.Output
		}
	}

	return output
}

// GetStats 获取执行器统计信息
func (e *executor) GetStats() map[string]any {
	return map[string]any{
		"max_concurrency":   e.maxConcurrency,
		"available_workers": len(e.workerPool),
		"active_workers":    e.maxConcurrency - len(e.workerPool),
	}
}

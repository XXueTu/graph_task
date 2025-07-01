package web

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// CreateWorkflowRequest 创建工作流请求
type CreateWorkflowRequest struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Version      string                 `json:"version"`
	Tasks        []TaskDefinition       `json:"tasks"`
	Dependencies []DependencyDefinition `json:"dependencies"`
}

// TaskDefinition 任务定义
type TaskDefinition struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Timeout     int    `json:"timeout"` // 超时时间(秒)
}

// DependencyDefinition 依赖定义
type DependencyDefinition struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ExecutionRequest 执行请求
type ExecutionRequest struct {
	WorkflowID string                 `json:"workflow_id"`
	Input      map[string]interface{} `json:"input"`
	Async      bool                   `json:"async"`
}

// WorkflowResponse 工作流响应
type WorkflowResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	TaskCount   int       `json:"task_count"`
}

// ExecutionResponse 执行响应
type ExecutionResponse struct {
	ID         string                 `json:"id"`
	WorkflowID string                 `json:"workflow_id"`
	Status     string                 `json:"status"`
	Input      map[string]interface{} `json:"input"`
	Output     map[string]interface{} `json:"output"`
	StartedAt  time.Time              `json:"started_at"`
	EndedAt    *time.Time             `json:"ended_at"`
	Duration   int64                  `json:"duration"` // 毫秒
	RetryCount int                    `json:"retry_count"`
}

// ExecutionStatusResponse 执行状态响应
type ExecutionStatusResponse struct {
	ExecutionID    string  `json:"execution_id"`
	Status         string  `json:"status"`
	Progress       float64 `json:"progress"`
	CompletedTasks int     `json:"completed_tasks"`
	TotalTasks     int     `json:"total_tasks"`
	CurrentTask    string  `json:"current_task"`
}

// LogEntryResponse 日志条目响应
type LogEntryResponse struct {
	ID          string    `json:"id"`
	ExecutionID string    `json:"execution_id"`
	TaskID      string    `json:"task_id"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
}

// RetryInfoResponse 重试信息响应
type RetryInfoResponse struct {
	ExecutionID   string    `json:"execution_id"`
	WorkflowID    string    `json:"workflow_id"`
	FailureReason string    `json:"failure_reason"`
	RetryCount    int       `json:"retry_count"`
	LastRetryAt   time.Time `json:"last_retry_at"`
	Status        string    `json:"status"`
}

// listWorkflows 列出工作流
func (s *Server) listWorkflows(w http.ResponseWriter, r *http.Request) {
	// limit := s.parseIntParam(r, "limit", 20)
	// offset := s.parseIntParam(r, "offset", 0)

	workflows, err := s.workflowService.ListWorkflows()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var response []WorkflowResponse
	for _, wf := range workflows {
		response = append(response, WorkflowResponse{
			ID:          wf.ID(),
			Name:        wf.Name(),
			Description: wf.Description(),
			Version:     wf.Version(),
			Status:      string(wf.Status()),
			CreatedAt:   wf.CreatedAt(),
			UpdatedAt:   wf.UpdatedAt(),
			TaskCount:   len(wf.Tasks()),
		})
	}

	s.writeJSON(w, http.StatusOK, response)
}

// createWorkflow 创建工作流
func (s *Server) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 验证请求
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "Workflow name is required")
		return
	}

	// 由于当前架构中工作流创建需要函数处理器，这里返回成功但提示需要通过SDK创建
	response := map[string]interface{}{
		"message": "Workflow definition received. Please use SDK to register with task handlers.",
		"template": map[string]interface{}{
			"name":         req.Name,
			"description":  req.Description,
			"version":      req.Version,
			"tasks":        req.Tasks,
			"dependencies": req.Dependencies,
		},
	}

	s.writeJSON(w, http.StatusAccepted, response)
}

// getWorkflow 获取工作流
func (s *Server) getWorkflow(w http.ResponseWriter, r *http.Request, workflowID string) {
	workflow, err := s.workflowService.GetWorkflow(workflowID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Workflow not found")
		return
	}

	response := WorkflowResponse{
		ID:          workflow.ID(),
		Name:        workflow.Name(),
		Description: workflow.Description(),
		Version:     workflow.Version(),
		Status:      string(workflow.Status()),
		CreatedAt:   workflow.CreatedAt(),
		UpdatedAt:   workflow.UpdatedAt(),
		TaskCount:   len(workflow.Tasks()),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// updateWorkflow 更新工作流
func (s *Server) updateWorkflow(w http.ResponseWriter, r *http.Request, workflowID string) {
	s.writeError(w, http.StatusNotImplemented, "Workflow update not supported via API. Use SDK instead.")
}

// deleteWorkflow 删除工作流
func (s *Server) deleteWorkflow(w http.ResponseWriter, r *http.Request, workflowID string) {
	err := s.workflowService.DeleteWorkflow(workflowID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Workflow deleted successfully"})
}

// listExecutions 列出执行记录
func (s *Server) listExecutions(w http.ResponseWriter, r *http.Request) {
	workflowID := r.URL.Query().Get("workflow_id")
	limit := s.parseIntParam(r, "limit", 20)
	offset := s.parseIntParam(r, "offset", 0)

	executions, err := s.executionService.ListExecutions(workflowID, limit, offset)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var response []ExecutionResponse
	for _, exec := range executions {
		var endedAt *time.Time
		if exec.IsCompleted() {
			endTime := exec.EndTime()
			endedAt = &endTime
		}

		response = append(response, ExecutionResponse{
			ID:         exec.ID(),
			WorkflowID: exec.WorkflowID(),
			Status:     string(exec.Status()),
			Input:      exec.Input(),
			Output:     exec.Output(),
			StartedAt:  exec.StartTime(),
			EndedAt:    endedAt,
			Duration:   exec.Duration().Milliseconds(),
			RetryCount: exec.RetryCount(),
		})
	}

	s.writeJSON(w, http.StatusOK, response)
}

// createExecution 创建执行
func (s *Server) createExecution(w http.ResponseWriter, r *http.Request) {
	var req ExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := context.Background()

	if req.Async {
		// 异步执行
		executionID, err := s.executionService.ExecuteAsync(ctx, req.WorkflowID, req.Input)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		response := map[string]interface{}{
			"execution_id": executionID,
			"async":        true,
			"message":      "Execution started asynchronously",
		}
		s.writeJSON(w, http.StatusAccepted, response)
	} else {
		// 同步执行
		execution, err := s.executionService.Execute(ctx, req.WorkflowID, req.Input)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		var endedAt *time.Time
		if execution.IsCompleted() {
			endTime := execution.EndTime()
			endedAt = &endTime
		}

		response := ExecutionResponse{
			ID:         execution.ID(),
			WorkflowID: execution.WorkflowID(),
			Status:     string(execution.Status()),
			Input:      execution.Input(),
			Output:     execution.Output(),
			StartedAt:  execution.StartTime(),
			EndedAt:    endedAt,
			Duration:   execution.Duration().Milliseconds(),
			RetryCount: execution.RetryCount(),
		}
		s.writeJSON(w, http.StatusOK, response)
	}
}

// getExecution 获取执行
func (s *Server) getExecution(w http.ResponseWriter, r *http.Request, executionID string) {
	execution, err := s.executionService.GetExecution(executionID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Execution not found")
		return
	}

	var endedAt *time.Time
	if execution.IsCompleted() {
		endTime := execution.EndTime()
		endedAt = &endTime
	}

	response := ExecutionResponse{
		ID:         execution.ID(),
		WorkflowID: execution.WorkflowID(),
		Status:     string(execution.Status()),
		Input:      execution.Input(),
		Output:     execution.Output(),
		StartedAt:  execution.StartTime(),
		EndedAt:    endedAt,
		Duration:   execution.Duration().Milliseconds(),
		RetryCount: execution.RetryCount(),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// cancelExecution 取消执行
func (s *Server) cancelExecution(w http.ResponseWriter, r *http.Request, executionID string) {
	err := s.executionService.CancelExecution(executionID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Execution cancelled successfully"})
}

// getExecutionStatus 获取执行状态
func (s *Server) getExecutionStatus(w http.ResponseWriter, r *http.Request, executionID string) {
	status, err := s.executionService.GetExecutionStatus(executionID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Execution status not found")
		return
	}

	response := ExecutionStatusResponse{
		ExecutionID:    executionID,
		Status:         string(status.Status()),
		Progress:       status.Progress(),
		CompletedTasks: status.CompletedTasks(),
		TotalTasks:     status.TotalTasks(),
		CurrentTask:    "",
	}

	s.writeJSON(w, http.StatusOK, response)
}

// getExecutionLogs 获取执行日志
func (s *Server) getExecutionLogs(w http.ResponseWriter, r *http.Request, executionID string) {
	limit := s.parseIntParam(r, "limit", 100)
	offset := s.parseIntParam(r, "offset", 0)

	logs, err := s.executionService.GetExecutionLogs(executionID, limit, offset)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var response []LogEntryResponse
	for _, log := range logs {
		response = append(response, LogEntryResponse{
			ID:          log.ID(),
			ExecutionID: log.ExecutionID(),
			TaskID:      log.TaskID(),
			Level:       string(log.Level()),
			Message:     log.Message(),
			Timestamp:   log.Timestamp(),
		})
	}

	s.writeJSON(w, http.StatusOK, response)
}

// getExecutionTrace 获取执行追踪
func (s *Server) getExecutionTrace(w http.ResponseWriter, r *http.Request, executionID string) {
	trace, err := s.executionService.GetExecutionTrace(executionID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Execution trace not found")
		return
	}

	// 简化的追踪响应
	response := map[string]interface{}{
		"execution_id": executionID,
		"trace_id":     trace.TraceID(),
		"spans":        len(trace.Spans()), // 跨度
		// "duration":     trace.Duration().Milliseconds(), // 毫秒
		// "status":       trace.Status(),                  // 状态
	}

	s.writeJSON(w, http.StatusOK, response)
}

// listRetries 列出重试信息
func (s *Server) listRetries(w http.ResponseWriter, r *http.Request) {
	// limit := s.parseIntParam(r, "limit", 20)
	// offset := s.parseIntParam(r, "offset", 0)

	// retries, err := s.retryService.ListFailedExecutions()
	// if err != nil {
	// 	s.writeError(w, http.StatusInternalServerError, err.Error())
	// 	return
	// }

	// var response []RetryInfoResponse
	// for _, retry := range retries {
	// 	response = append(response, RetryInfoResponse{
	// 		ExecutionID:   retry.ExecutionID(),
	// 		WorkflowID:    retry.WorkflowID(),
	// 		FailureReason: retry.FailureReason(),
	// 		RetryCount:    retry.RetryCount(),
	// 		LastRetryAt:   retry.LastRetryAt(),
	// 		Status:        string(retry.Status()),
	// 	})
	// }

	// s.writeJSON(w, http.StatusOK, response)
}

// getRetry 获取重试信息
func (s *Server) getRetry(w http.ResponseWriter, r *http.Request, executionID string) {
	retry, err := s.retryService.GetRetryInfo(executionID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Retry info not found")
		return
	}

	response := RetryInfoResponse{
		ExecutionID:   retry.ExecutionID(),
		WorkflowID:    retry.WorkflowID(),
		FailureReason: retry.FailureReason(),
		RetryCount:    retry.RetryCount(),
		LastRetryAt:   *retry.LastRetryAt(),
		Status:        string(retry.Status()),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// manualRetry 手动重试
func (s *Server) manualRetry(w http.ResponseWriter, r *http.Request, executionID string) {
	// ctx := context.Background()

	// execution, err := s.retryService.ManualRetry(ctx, executionID)
	// if err != nil {
	// 	s.writeError(w, http.StatusInternalServerError, err.Error())
	// 	return
	// }

	// var endedAt *time.Time
	// if execution.IsCompleted() {
	// 	endTime := execution.EndTime()
	// 	endedAt = &endTime
	// }

	// response := ExecutionResponse{
	// 	ID:         execution.ExecutionID(),
	// 	WorkflowID: execution.WorkflowID(),
	// 	Status:     string(execution.Status()),
	// 	Input:      execution.Input(),
	// 	Output:     execution.Output(),
	// 	StartedAt:  execution.StartedAt(),
	// 	EndedAt:    endedAt,
	// 	Duration:   execution.Duration().Milliseconds(),
	// 	RetryCount: execution.RetryCount(),
	// }

	// s.writeJSON(w, http.StatusOK, response)
}

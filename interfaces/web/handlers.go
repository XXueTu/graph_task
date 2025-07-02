package web

import (
	"net/http"
	"time"
)

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

// TaskResultResponse 任务结果响应
type TaskResultResponse struct {
	ID          string                 `json:"id"`
	ExecutionID string                 `json:"execution_id"`
	TaskID      string                 `json:"task_id"`
	Status      string                 `json:"status"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output"`
	Error       string                 `json:"error"`
	StartTime   *time.Time             `json:"start_time"`
	EndTime     *time.Time             `json:"end_time"`
	Duration    int64                  `json:"duration"` // 毫秒
	RetryCount  int                    `json:"retry_count"`
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
			Status:     exec.Status().String(),
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

// getExecutionTaskResults 获取执行的任务结果列表
func (s *Server) getExecutionTaskResults(w http.ResponseWriter, r *http.Request, executionID string) {
	taskResults, err := s.executionService.GetExecutionTaskResults(executionID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var response []TaskResultResponse
	for _, result := range taskResults {
		var startTime, endTime *time.Time
		if !result.StartTime().IsZero() {
			st := result.StartTime()
			startTime = &st
		}
		if !result.EndTime().IsZero() {
			et := result.EndTime()
			endTime = &et
		}

		response = append(response, TaskResultResponse{
			ID:          result.ID(),
			ExecutionID: executionID,  // Use the executionID parameter
			TaskID:      result.TaskID(),
			Status:      result.Status().String(),
			Input:       result.Input(),
			Output:      result.Output(),
			Error:       result.Error(),
			StartTime:   startTime,
			EndTime:     endTime,
			Duration:    result.Duration().Milliseconds(),
			RetryCount:  result.RetryCount(),
		})
	}

	s.writeJSON(w, http.StatusOK, response)
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
		Status:     execution.Status().String(),
		Input:      execution.Input(),
		Output:     execution.Output(),
		StartedAt:  execution.StartTime(),
		EndedAt:    endedAt,
		Duration:   execution.Duration().Milliseconds(),
		RetryCount: execution.RetryCount(),
	}

	s.writeJSON(w, http.StatusOK, response)
}
package execution

import (
	"sync"
	"time"
)

// ExecutionStatus 执行状态信息
type ExecutionStatus struct {
	executionID   string
	workflowID    string
	status        Status
	progress      float64
	totalTasks    int
	completedTasks int
	failedTasks   int
	startTime     time.Time
	lastUpdateTime time.Time
	estimatedEnd  *time.Time
	currentStage  string
}

// NewExecutionStatus 创建执行状态
func NewExecutionStatus(executionID, workflowID string, totalTasks int) *ExecutionStatus {
	now := time.Now()
	return &ExecutionStatus{
		executionID:    executionID,
		workflowID:     workflowID,
		status:         StatusPending,
		progress:       0.0,
		totalTasks:     totalTasks,
		completedTasks: 0,
		failedTasks:    0,
		startTime:      now,
		lastUpdateTime: now,
		currentStage:   "pending",
	}
}

// ExecutionStatus getter methods
func (s *ExecutionStatus) ExecutionID() string { return s.executionID }
func (s *ExecutionStatus) WorkflowID() string { return s.workflowID }
func (s *ExecutionStatus) Status() Status { return s.status }
func (s *ExecutionStatus) Progress() float64 { return s.progress }
func (s *ExecutionStatus) TotalTasks() int { return s.totalTasks }
func (s *ExecutionStatus) CompletedTasks() int { return s.completedTasks }
func (s *ExecutionStatus) FailedTasks() int { return s.failedTasks }
func (s *ExecutionStatus) StartTime() time.Time { return s.startTime }
func (s *ExecutionStatus) LastUpdateTime() time.Time { return s.lastUpdateTime }
func (s *ExecutionStatus) EstimatedEnd() *time.Time { return s.estimatedEnd }
func (s *ExecutionStatus) CurrentStage() string { return s.currentStage }

// UpdateProgress 更新进度
func (s *ExecutionStatus) UpdateProgress(completedTasks, failedTasks int, currentStage string) {
	s.completedTasks = completedTasks
	s.failedTasks = failedTasks
	s.currentStage = currentStage
	s.lastUpdateTime = time.Now()
	
	if s.totalTasks > 0 {
		s.progress = float64(completedTasks) / float64(s.totalTasks) * 100.0
	}
	
	// 估算结束时间
	if completedTasks > 0 && s.progress > 0 {
		elapsed := time.Since(s.startTime)
		estimatedTotal := time.Duration(float64(elapsed) / s.progress * 100.0)
		estimatedEnd := s.startTime.Add(estimatedTotal)
		s.estimatedEnd = &estimatedEnd
	}
}

// SetStatus 设置状态
func (s *ExecutionStatus) SetStatus(status Status) {
	s.status = status
	s.lastUpdateTime = time.Now()
}

// IsCompleted 是否已完成
func (s *ExecutionStatus) IsCompleted() bool {
	return s.status == StatusSuccess || s.status == StatusFailed || s.status == StatusCanceled
}

// MonitorService 执行监控服务
type MonitorService interface {
	// StartMonitoring 开始监控执行
	StartMonitoring(executionID, workflowID string, totalTasks int) *ExecutionStatus
	
	// UpdateExecutionProgress 更新执行进度
	UpdateExecutionProgress(executionID string, completedTasks, failedTasks int, currentStage string)
	
	// SetExecutionStatus 设置执行状态
	SetExecutionStatus(executionID string, status Status)
	
	// GetExecutionStatus 获取执行状态
	GetExecutionStatus(executionID string) (*ExecutionStatus, error)
	
	// ListRunningExecutions 列出正在运行的执行
	ListRunningExecutions() []*ExecutionStatus
	
	// StopMonitoring 停止监控
	StopMonitoring(executionID string)
	
	// CleanupOldStatuses 清理旧的状态
	CleanupOldStatuses(maxAge time.Duration)
}

// monitorService 监控服务实现
type monitorService struct {
	statuses map[string]*ExecutionStatus
	mutex    sync.RWMutex
}

// NewMonitorService 创建监控服务
func NewMonitorService() MonitorService {
	return &monitorService{
		statuses: make(map[string]*ExecutionStatus),
	}
}

// StartMonitoring 开始监控执行
func (m *monitorService) StartMonitoring(executionID, workflowID string, totalTasks int) *ExecutionStatus {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	status := NewExecutionStatus(executionID, workflowID, totalTasks)
	m.statuses[executionID] = status
	
	return status
}

// UpdateExecutionProgress 更新执行进度
func (m *monitorService) UpdateExecutionProgress(executionID string, completedTasks, failedTasks int, currentStage string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if status, exists := m.statuses[executionID]; exists {
		status.UpdateProgress(completedTasks, failedTasks, currentStage)
	}
}

// SetExecutionStatus 设置执行状态
func (m *monitorService) SetExecutionStatus(executionID string, status Status) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if execStatus, exists := m.statuses[executionID]; exists {
		execStatus.SetStatus(status)
	}
}

// GetExecutionStatus 获取执行状态
func (m *monitorService) GetExecutionStatus(executionID string) (*ExecutionStatus, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	status, exists := m.statuses[executionID]
	if !exists {
		return nil, NewExecutionError("execution status not found: " + executionID)
	}
	
	return status, nil
}

// ListRunningExecutions 列出正在运行的执行
func (m *monitorService) ListRunningExecutions() []*ExecutionStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	var running []*ExecutionStatus
	for _, status := range m.statuses {
		if status.Status() == StatusRunning {
			running = append(running, status)
		}
	}
	
	return running
}

// StopMonitoring 停止监控
func (m *monitorService) StopMonitoring(executionID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	delete(m.statuses, executionID)
}

// CleanupOldStatuses 清理旧的状态
func (m *monitorService) CleanupOldStatuses(maxAge time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	now := time.Now()
	for executionID, status := range m.statuses {
		if status.IsCompleted() && now.Sub(status.LastUpdateTime()) > maxAge {
			delete(m.statuses, executionID)
		}
	}
}
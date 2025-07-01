package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/XXueTu/graph_task/domain/retry"
)

// RetryService 重试应用服务
type RetryService struct {
	retryRepo        retry.Repository
	executionService *ExecutionService // 引用执行服务进行重试
	eventPub         EventPublisher
}

// NewRetryService 创建重试服务
func NewRetryService(retryRepo retry.Repository, eventPub EventPublisher) *RetryService {
	return &RetryService{
		retryRepo: retryRepo,
		eventPub:  eventPub,
	}
}

// SetExecutionService 设置执行服务（用于循环依赖）
func (s *RetryService) SetExecutionService(executionService *ExecutionService) {
	s.executionService = executionService
}

// ManualRetry 手动重试失败的执行
func (s *RetryService) ManualRetry(ctx context.Context, executionID string) (*retry.ManualRetryRecord, error) {
	// 获取重试信息
	retryInfo, err := s.retryRepo.FindRetryInfo(executionID)
	if err != nil {
		return nil, NewApplicationErrorf("retry info not found: %s", executionID)
	}

	if !retryInfo.CanRetry() {
		return nil, NewApplicationError("execution cannot be retried")
	}

	// 生成新的尝试ID
	attemptID := s.generateAttemptID(executionID)

	// 发布重试开始事件
	event := NewRetryEvent("retry.started", executionID, retryInfo.WorkflowID(), map[string]interface{}{
		"attempt_id": attemptID,
	})
	s.eventPub.Publish(event)

	// 执行重试
	var record *retry.ManualRetryRecord
	if s.executionService != nil {
		result, err := s.executionService.Execute(ctx, retryInfo.WorkflowID(), retryInfo.Input())

		if err != nil {
			// 重试失败
			record = retry.NewManualRetryRecord(attemptID, "", false, err.Error())
			event = NewRetryEvent("retry.failed", executionID, retryInfo.WorkflowID(), map[string]interface{}{
				"attempt_id": attemptID,
				"error":      err.Error(),
			})
		} else {
			// 重试成功
			record = retry.NewManualRetryRecord(attemptID, result.ID(), true, "")
			event = NewRetryEvent("retry.succeeded", executionID, retryInfo.WorkflowID(), map[string]interface{}{
				"attempt_id":       attemptID,
				"new_execution_id": result.ID(),
			})
		}
	} else {
		return nil, NewApplicationError("execution service not available")
	}

	// 记录手动重试
	retryInfo.AddManualRetry(record)
	if saveErr := s.retryRepo.UpdateRetryInfo(retryInfo); saveErr != nil {
		fmt.Printf("Failed to update retry info: %v\n", saveErr)
	}

	// 发布重试结果事件
	s.eventPub.Publish(event)

	return record, nil
}

// GetFailedExecutions 获取失败的执行
func (s *RetryService) GetFailedExecutions() ([]*retry.Info, error) {
	return s.retryRepo.ListFailedExecutions()
}

// GetRetryInfo 获取重试信息
func (s *RetryService) GetRetryInfo(executionID string) (*retry.Info, error) {
	return s.retryRepo.FindRetryInfo(executionID)
}

// GetRetryStatistics 获取重试统计
func (s *RetryService) GetRetryStatistics() (*retry.Statistics, error) {
	return s.retryRepo.GetStatistics()
}

// AbandonRetry 放弃重试
func (s *RetryService) AbandonRetry(executionID string) error {
	retryInfo, err := s.retryRepo.FindRetryInfo(executionID)
	if err != nil {
		return err
	}

	retryInfo.MarkAbandoned()
	if err := s.retryRepo.UpdateRetryInfo(retryInfo); err != nil {
		return err
	}

	// 发布放弃重试事件
	event := NewRetryEvent("retry.abandoned", executionID, retryInfo.WorkflowID(), map[string]interface{}{})
	s.eventPub.Publish(event)

	return nil
}

// RecordFailedExecution 记录失败执行
func (s *RetryService) RecordFailedExecution(executionID, workflowID string, input map[string]interface{}, err error) error {
	retryInfo := retry.NewInfo(executionID, workflowID, input, err.Error())
	return s.retryRepo.SaveRetryInfo(retryInfo)
}

// generateAttemptID 生成尝试ID
func (s *RetryService) generateAttemptID(executionID string) string {
	// 简化实现
	uuid := uuid.New()
	return fmt.Sprintf("%s_retry_%s", executionID, uuid.String())
}

// getRetryInfo 内部方法获取重试信息
func (s *RetryService) getRetryInfo(executionID string) *retry.Info {
	info, _ := s.retryRepo.FindRetryInfo(executionID)
	if info == nil {
		// 返回空的重试信息
		return retry.NewInfo(executionID, "", nil, "")
	}
	return info
}

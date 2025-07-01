package engine

import (
	"fmt"
	"sync"
	"time"

	"github.com/XXueTu/graph_task/types"
)

// RetryManager 重试管理器
type RetryManager struct {
	config      *RetryConfig
	failedExecs map[string]*RetryInfo
	mutex       sync.RWMutex
	storage     RetryStorage // 可选的持久化存储
}

type RetryConfig struct {
	MaxAutoRetries    int           // 最大自动重试次数
	RetryDelay        time.Duration // 重试延迟
	BackoffMultiplier float64       // 退避倍数
	MaxRetryDelay     time.Duration // 最大重试延迟
}

// Type aliases
type RetryInfo = types.RetryInfo
type ManualRetryRecord = types.ManualRetryRecord

// Constants for retry status
const (
	RetryStatusPending   = "pending"   // 待重试
	RetryStatusRetrying  = "retrying"  // 重试中
	RetryStatusExhausted = "exhausted" // 自动重试已耗尽
	RetryStatusAbandoned = "abandoned" // 已放弃
	RetryStatusRecovered = "recovered" // 已恢复
)

// RetryStorage 重试存储接口
type RetryStorage interface {
	SaveRetryInfo(info *RetryInfo) error
	GetRetryInfo(executionID string) (*RetryInfo, error)
	ListFailedExecutions() ([]*RetryInfo, error)
	UpdateRetryInfo(info *RetryInfo) error
	DeleteRetryInfo(executionID string) error
}

// NewRetryManager 创建重试管理器
func NewRetryManager(config *RetryConfig, storage ...RetryStorage) *RetryManager {
	rm := &RetryManager{
		config:      config,
		failedExecs: make(map[string]*RetryInfo),
	}

	if len(storage) > 0 {
		rm.storage = storage[0]
	}

	return rm
}

// RecordFailedExecution 记录失败的执行
func (rm *RetryManager) RecordFailedExecution(executionID, workflowID string, input map[string]any, err error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	retryInfo := &RetryInfo{
		ExecutionID:   executionID,
		WorkflowID:    workflowID,
		Input:         input,
		FailureReason: err.Error(),
		FailedAt:      time.Now(),
		RetryCount:    rm.config.MaxAutoRetries,
		Status:        RetryStatusExhausted,
		ManualRetries: make([]ManualRetryRecord, 0),
	}

	rm.failedExecs[executionID] = retryInfo

	// 持久化
	if rm.storage != nil {
		rm.storage.SaveRetryInfo(retryInfo)
	}
}

// GetRetryInfo 获取重试信息
func (rm *RetryManager) GetRetryInfo(executionID string) (*RetryInfo, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	if info, exists := rm.failedExecs[executionID]; exists {
		return info, nil
	}

	// 从存储中获取
	if rm.storage != nil {
		return rm.storage.GetRetryInfo(executionID)
	}

	return nil, fmt.Errorf("retry info not found for execution: %s", executionID)
}

// GetFailedExecutions 获取所有失败的执行
func (rm *RetryManager) GetFailedExecutions() ([]*RetryInfo, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	var failed []*RetryInfo
	for _, info := range rm.failedExecs {
		if info.Status == RetryStatusExhausted || info.Status == RetryStatusPending {
			failed = append(failed, info)
		}
	}

	// 从存储中获取更多
	if rm.storage != nil {
		if storedFailed, err := rm.storage.ListFailedExecutions(); err == nil {
			failed = append(failed, storedFailed...)
		}
	}

	return failed, nil
}

// RecordManualRetry 记录手动重试
func (rm *RetryManager) RecordManualRetry(originalExecutionID, newExecutionID string, success bool, errorMsg string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	info, exists := rm.failedExecs[originalExecutionID]
	if !exists {
		return fmt.Errorf("retry info not found for execution: %s", originalExecutionID)
	}

	record := ManualRetryRecord{
		AttemptID:    generateRetryAttemptID(),
		RetryAt:      time.Now(),
		Success:      success,
		ErrorMessage: errorMsg,
		ExecutionID:  newExecutionID,
	}

	info.ManualRetries = append(info.ManualRetries, record)
	info.LastRetryAt = &record.RetryAt

	if success {
		info.Status = RetryStatusRecovered
	}

	// 持久化
	if rm.storage != nil {
		rm.storage.UpdateRetryInfo(info)
	}

	return nil
}

// AbandonRetry 放弃重试
func (rm *RetryManager) AbandonRetry(executionID string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	info, exists := rm.failedExecs[executionID]
	if !exists {
		return fmt.Errorf("retry info not found for execution: %s", executionID)
	}

	info.Status = RetryStatusAbandoned

	// 持久化
	if rm.storage != nil {
		rm.storage.UpdateRetryInfo(info)
	}

	return nil
}

// CleanupOldRetries 清理旧的重试记录
func (rm *RetryManager) CleanupOldRetries(olderThan time.Duration) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	cutoff := time.Now().Add(-olderThan)

	for executionID, info := range rm.failedExecs {
		if info.FailedAt.Before(cutoff) &&
			(info.Status == RetryStatusRecovered || info.Status == RetryStatusAbandoned) {
			delete(rm.failedExecs, executionID)

			// 从存储中删除
			if rm.storage != nil {
				rm.storage.DeleteRetryInfo(executionID)
			}
		}
	}
}

// GetRetryStatistics 获取重试统计
func (rm *RetryManager) GetRetryStatistics() *RetryStatistics {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	stats := &RetryStatistics{
		TotalFailed:   0,
		PendingRetry:  0,
		Recovered:     0,
		Abandoned:     0,
		ManualRetries: 0,
	}

	for _, info := range rm.failedExecs {
		stats.TotalFailed++

		switch info.Status {
		case RetryStatusPending, RetryStatusExhausted:
			stats.PendingRetry++
		case RetryStatusRecovered:
			stats.Recovered++
		case RetryStatusAbandoned:
			stats.Abandoned++
		}

		stats.ManualRetries += len(info.ManualRetries)
	}

	return stats
}

// RetryStatistics 重试统计
type RetryStatistics struct {
	TotalFailed   int `json:"total_failed"`
	PendingRetry  int `json:"pending_retry"`
	Recovered     int `json:"recovered"`
	Abandoned     int `json:"abandoned"`
	ManualRetries int `json:"manual_retries"`
}

// 辅助函数
func generateRetryAttemptID() string {
	return fmt.Sprintf("retry_%d", time.Now().UnixNano())
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAutoRetries:    3,
		RetryDelay:        1 * time.Second,
		BackoffMultiplier: 2.0,
		MaxRetryDelay:     30 * time.Second,
	}
}

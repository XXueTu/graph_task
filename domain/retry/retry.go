package retry

import (
	"time"
)

// Status 重试状态
type Status string

const (
	StatusPending    Status = "pending"    // 等待重试
	StatusExhausted  Status = "exhausted"  // 重试耗尽
	StatusAbandoned  Status = "abandoned"  // 放弃重试
	StatusRecovered  Status = "recovered"  // 已恢复
)

// ManualRetryRecord 手动重试记录
type ManualRetryRecord struct {
	attemptID    string
	retryAt      time.Time
	success      bool
	errorMessage string
	executionID  string // 新的执行ID
}

// NewManualRetryRecord 创建手动重试记录
func NewManualRetryRecord(attemptID, executionID string, success bool, errorMessage string) *ManualRetryRecord {
	return &ManualRetryRecord{
		attemptID:    attemptID,
		retryAt:      time.Now(),
		success:      success,
		errorMessage: errorMessage,
		executionID:  executionID,
	}
}

// ManualRetryRecord getter methods
func (r *ManualRetryRecord) AttemptID() string { return r.attemptID }
func (r *ManualRetryRecord) RetryAt() time.Time { return r.retryAt }
func (r *ManualRetryRecord) Success() bool { return r.success }
func (r *ManualRetryRecord) ErrorMessage() string { return r.errorMessage }
func (r *ManualRetryRecord) ExecutionID() string { return r.executionID }

// Info 重试信息聚合根
type Info struct {
	executionID   string
	workflowID    string
	input         map[string]interface{}
	failureReason string
	failedAt      time.Time
	retryCount    int
	lastRetryAt   *time.Time
	status        Status
	manualRetries []*ManualRetryRecord
}

// NewInfo 创建重试信息
func NewInfo(executionID, workflowID string, input map[string]interface{}, failureReason string) *Info {
	return &Info{
		executionID:   executionID,
		workflowID:    workflowID,
		input:         input,
		failureReason: failureReason,
		failedAt:      time.Now(),
		status:        StatusPending,
		manualRetries: make([]*ManualRetryRecord, 0),
	}
}

// Info getter methods
func (i *Info) ExecutionID() string { return i.executionID }
func (i *Info) WorkflowID() string { return i.workflowID }
func (i *Info) Input() map[string]interface{} { return i.input }
func (i *Info) FailureReason() string { return i.failureReason }
func (i *Info) FailedAt() time.Time { return i.failedAt }
func (i *Info) RetryCount() int { return i.retryCount }
func (i *Info) LastRetryAt() *time.Time { return i.lastRetryAt }
func (i *Info) Status() Status { return i.status }
func (i *Info) ManualRetries() []*ManualRetryRecord { return i.manualRetries }

// IncrementRetryCount 增加重试次数
func (i *Info) IncrementRetryCount() {
	i.retryCount++
	now := time.Now()
	i.lastRetryAt = &now
}

// MarkExhausted 标记为重试耗尽
func (i *Info) MarkExhausted() {
	i.status = StatusExhausted
}

// MarkAbandoned 标记为放弃重试
func (i *Info) MarkAbandoned() {
	i.status = StatusAbandoned
}

// MarkRecovered 标记为已恢复
func (i *Info) MarkRecovered() {
	i.status = StatusRecovered
}

// AddManualRetry 添加手动重试记录
func (i *Info) AddManualRetry(record *ManualRetryRecord) {
	i.manualRetries = append(i.manualRetries, record)
	if record.Success() {
		i.MarkRecovered()
	}
}

// CanRetry 是否可以重试
func (i *Info) CanRetry() bool {
	return i.status == StatusPending || i.status == StatusExhausted
}

// Config 重试配置
type Config struct {
	maxAutoRetries    int
	retryDelay        time.Duration
	backoffMultiplier float64
	maxRetryDelay     time.Duration
}

// NewConfig 创建重试配置
func NewConfig(maxAutoRetries int, retryDelay time.Duration, backoffMultiplier float64, maxRetryDelay time.Duration) *Config {
	return &Config{
		maxAutoRetries:    maxAutoRetries,
		retryDelay:        retryDelay,
		backoffMultiplier: backoffMultiplier,
		maxRetryDelay:     maxRetryDelay,
	}
}

// DefaultConfig 默认重试配置
func DefaultConfig() *Config {
	return NewConfig(3, time.Second, 2.0, 30*time.Second)
}

// Config getter methods
func (c *Config) MaxAutoRetries() int { return c.maxAutoRetries }
func (c *Config) RetryDelay() time.Duration { return c.retryDelay }
func (c *Config) BackoffMultiplier() float64 { return c.backoffMultiplier }
func (c *Config) MaxRetryDelay() time.Duration { return c.maxRetryDelay }

// CalculateDelay 计算退避延迟
func (c *Config) CalculateDelay(attempt int) time.Duration {
	delay := time.Duration(float64(c.retryDelay.Nanoseconds()) * 
		pow(c.backoffMultiplier, float64(attempt-1)))
	
	if delay > c.maxRetryDelay {
		delay = c.maxRetryDelay
	}
	
	return delay
}

// pow 简单的幂运算
func pow(base, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	result := base
	for i := 1; i < int(exp); i++ {
		result *= base
	}
	return result
}

// Statistics 重试统计
type Statistics struct {
	totalFailed   int
	recovered     int
	pendingRetry  int
	exhausted     int
	abandoned     int
}

// NewStatistics 创建重试统计
func NewStatistics(totalFailed, recovered, pendingRetry, exhausted, abandoned int) *Statistics {
	return &Statistics{
		totalFailed:   totalFailed,
		recovered:     recovered,
		pendingRetry:  pendingRetry,
		exhausted:     exhausted,
		abandoned:     abandoned,
	}
}

// Statistics getter methods
func (s *Statistics) TotalFailed() int { return s.totalFailed }
func (s *Statistics) Recovered() int { return s.recovered }
func (s *Statistics) PendingRetry() int { return s.pendingRetry }
func (s *Statistics) Exhausted() int { return s.exhausted }
func (s *Statistics) Abandoned() int { return s.abandoned }
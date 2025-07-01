package logger

import (
	"context"
	"sync"
	"time"
)

type LogContextKey string

// 自定义context的key
const (
	ExecutionIDKey LogContextKey = "executionId"
	TaskIDKey      LogContextKey = "taskId"
)

// LogLevel 日志级别
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// LogEntry 日志条目
type LogEntry struct {
	id          string
	executionID string
	taskID      string
	level       LogLevel
	message     string
	attributes  map[string]interface{}
	timestamp   time.Time
}

// NewLogEntry 创建日志条目
func NewLogEntry(executionID, taskID string, level LogLevel, message string, attributes map[string]interface{}) *LogEntry {
	return &LogEntry{
		id:          generateLogID(),
		executionID: executionID,
		taskID:      taskID,
		level:       level,
		message:     message,
		attributes:  attributes,
		timestamp:   time.Now(),
	}
}

// LogEntry getter methods
func (l *LogEntry) ID() string                         { return l.id }
func (l *LogEntry) ExecutionID() string                { return l.executionID }
func (l *LogEntry) TaskID() string                     { return l.taskID }
func (l *LogEntry) Level() LogLevel                    { return l.level }
func (l *LogEntry) Message() string                    { return l.message }
func (l *LogEntry) Attributes() map[string]interface{} { return l.attributes }
func (l *LogEntry) Timestamp() time.Time               { return l.timestamp }

// LoggerService 执行日志服务
type LoggerService interface {
	// Debug 记录调试日志
	Debug(ctx context.Context, message string, attributes map[string]interface{})

	// Info 记录信息日志
	Info(ctx context.Context, message string, attributes map[string]interface{})

	// Warn 记录警告日志
	Warn(ctx context.Context, message string, attributes map[string]interface{})

	// Error 记录错误日志
	Error(ctx context.Context, message string, attributes map[string]interface{})

	// GetLogs 获取执行日志
	GetLogs(executionID string, limit, offset int) ([]*LogEntry, error)

	// GetTaskLogs 获取任务日志
	GetTaskLogs(executionID, taskID string, limit, offset int) ([]*LogEntry, error)

	// FlushLogs 批量刷新日志
	FlushLogs() error

	// Close 关闭日志服务
	Close() error
}

// LogRepository 日志仓储接口
type LogRepository interface {
	// SaveLogs 批量保存日志
	SaveLogs(logs []*LogEntry) error

	// GetLogs 获取执行日志
	GetLogs(executionID string, limit, offset int) ([]*LogEntry, error)

	// GetTaskLogs 获取任务日志
	GetTaskLogs(executionID, taskID string, limit, offset int) ([]*LogEntry, error)

	// DeleteLogs 删除日志
	DeleteLogs(executionID string) error
}

// loggerService 日志服务实现
type loggerService struct {
	repository    LogRepository
	pendingLogs   []*LogEntry
	batchSize     int
	flushInterval time.Duration
	mutex         sync.Mutex
	stopCh        chan struct{}
}

// NewLoggerService 创建日志服务
func NewLoggerService(repository LogRepository, batchSize int, flushInterval time.Duration) LoggerService {
	service := &loggerService{
		repository:    repository,
		pendingLogs:   make([]*LogEntry, 0, batchSize),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		stopCh:        make(chan struct{}),
	}

	// 启动批量刷新协程
	go service.backgroundFlush()

	return service
}

// Debug 记录调试日志
func (l *loggerService) Debug(ctx context.Context, message string, attributes map[string]interface{}) {
	executionID, taskID := getExecutionIDAndTaskID(ctx)
	l.log(executionID, taskID, LogLevelDebug, message, attributes)
}

// Info 记录信息日志
func (l *loggerService) Info(ctx context.Context, message string, attributes map[string]interface{}) {
	executionID, taskID := getExecutionIDAndTaskID(ctx)
	l.log(executionID, taskID, LogLevelInfo, message, attributes)
}

// Warn 记录警告日志
func (l *loggerService) Warn(ctx context.Context, message string, attributes map[string]interface{}) {
	executionID, taskID := getExecutionIDAndTaskID(ctx)
	l.log(executionID, taskID, LogLevelWarn, message, attributes)
}

// Error 记录错误日志
func (l *loggerService) Error(ctx context.Context, message string, attributes map[string]interface{}) {
	executionID, taskID := getExecutionIDAndTaskID(ctx)
	l.log(executionID, taskID, LogLevelError, message, attributes)
}

func getExecutionIDAndTaskID(ctx context.Context) (string, string) {
	executionID := ctx.Value(ExecutionIDKey).(string)
	taskID := ctx.Value(TaskIDKey).(string)
	return executionID, taskID
}

// log 记录日志
func (l *loggerService) log(executionID, taskID string, level LogLevel, message string, attributes map[string]interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	entry := NewLogEntry(executionID, taskID, level, message, attributes)
	l.pendingLogs = append(l.pendingLogs, entry)

	// 如果达到批量大小，立即刷新
	if len(l.pendingLogs) >= l.batchSize {
		l.flushPendingLogs()
	}
}

// GetLogs 获取执行日志
func (l *loggerService) GetLogs(executionID string, limit, offset int) ([]*LogEntry, error) {
	return l.repository.GetLogs(executionID, limit, offset)
}

// GetTaskLogs 获取任务日志
func (l *loggerService) GetTaskLogs(executionID, taskID string, limit, offset int) ([]*LogEntry, error) {
	return l.repository.GetTaskLogs(executionID, taskID, limit, offset)
}

// FlushLogs 批量刷新日志
func (l *loggerService) FlushLogs() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	return l.flushPendingLogs()
}

// flushPendingLogs 刷新待处理的日志
func (l *loggerService) flushPendingLogs() error {
	if len(l.pendingLogs) == 0 {
		return nil
	}

	// 批量保存日志
	if err := l.repository.SaveLogs(l.pendingLogs); err != nil {
		return err
	}

	// 清空待刷新队列
	l.pendingLogs = l.pendingLogs[:0]

	return nil
}

// backgroundFlush 后台定期刷新
func (l *loggerService) backgroundFlush() {
	ticker := time.NewTicker(l.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.FlushLogs()
		case <-l.stopCh:
			// 最后一次刷新
			l.FlushLogs()
			return
		}
	}
}

// Close 关闭日志服务
func (l *loggerService) Close() error {
	close(l.stopCh)

	// 等待背景刷新完成
	time.Sleep(100 * time.Millisecond)

	return l.FlushLogs()
}

// generateLogID 生成日志ID
func generateLogID() string {
	return "log_" + time.Now().Format("20060102150405.000000")
}

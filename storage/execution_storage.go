package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/XXueTu/graph_task/types"
)


// ExecutionStorage 执行数据存储（仅存储执行记录，不存储工作流定义）
type ExecutionStorage interface {
	// 执行记录管理
	SaveExecution(result *types.ExecutionResult) error
	GetExecution(executionID string) (*types.ExecutionResult, error)
	UpdateExecution(result *types.ExecutionResult) error
	ListExecutions(workflowID string, offset, limit int) ([]*types.ExecutionResult, error)

	// 任务执行记录管理
	SaveTaskExecution(executionID string, result *types.TaskExecutionResult) error
	GetTaskExecution(executionID, taskID string) (*types.TaskExecutionResult, error)
	ListTaskExecutions(executionID string) ([]*types.TaskExecutionResult, error)

	// 重试记录管理
	SaveRetryInfo(info *types.RetryInfo) error
	GetRetryInfo(executionID string) (*types.RetryInfo, error)
	ListFailedExecutions() ([]*types.RetryInfo, error)
	UpdateRetryInfo(info *types.RetryInfo) error
	DeleteRetryInfo(executionID string) error

	// 追踪记录管理
	SaveTrace(trace *types.ExecutionTrace) error
	GetTrace(traceID string) (*types.ExecutionTrace, error)
	SaveSpan(span *types.TraceSpan) error
	GetSpan(spanID string) (*types.TraceSpan, error)
	GetTraceSpans(traceID string) ([]*types.TraceSpan, error)
	ListTraces(workflowID string, offset, limit int) ([]*types.ExecutionTrace, error)

	Close() error
}

// mysqlExecutionStorage MySQL执行存储实现
type mysqlExecutionStorage struct {
	db *sql.DB
}

// NewMySQLExecutionStorage 创建MySQL执行存储
func NewMySQLExecutionStorage(dsn string) (ExecutionStorage, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	storage := &mysqlExecutionStorage{db: db}

	// 初始化表结构
	if err := storage.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return storage, nil
}

// initTables 初始化数据库表（仅执行相关表）
func (s *mysqlExecutionStorage) initTables() error {
	queries := []string{
		// 执行记录表
		`CREATE TABLE IF NOT EXISTS executions (
			execution_id VARCHAR(255) PRIMARY KEY,
			workflow_id VARCHAR(255) NOT NULL,
			status INT NOT NULL DEFAULT 0,
			input JSON,
			output JSON,
			error TEXT,
			start_time TIMESTAMP NULL,
			end_time TIMESTAMP NULL,
			duration BIGINT,
			retry_count INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_workflow_id (workflow_id),
			INDEX idx_status (status),
			INDEX idx_start_time (start_time),
			INDEX idx_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		// 任务执行记录表
		`CREATE TABLE IF NOT EXISTS task_executions (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			execution_id VARCHAR(255) NOT NULL,
			task_id VARCHAR(255) NOT NULL,
			status INT NOT NULL DEFAULT 0,
			input JSON,
			output JSON,
			error TEXT,
			start_time TIMESTAMP NULL,
			end_time TIMESTAMP NULL,
			duration BIGINT,
			retry_count INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_execution_id (execution_id),
			INDEX idx_task_id (task_id),
			INDEX idx_status (status),
			INDEX idx_start_time (start_time),
			UNIQUE KEY uk_execution_task (execution_id, task_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		// 重试信息表
		`CREATE TABLE IF NOT EXISTS retry_info (
			execution_id VARCHAR(255) PRIMARY KEY,
			workflow_id VARCHAR(255) NOT NULL,
			input JSON,
			failure_reason TEXT,
			failed_at TIMESTAMP NOT NULL,
			retry_count INT DEFAULT 0,
			last_retry_at TIMESTAMP NULL,
			status VARCHAR(20) NOT NULL,
			manual_retries JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_workflow_id (workflow_id),
			INDEX idx_status (status),
			INDEX idx_failed_at (failed_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		// 执行追踪表
		`CREATE TABLE IF NOT EXISTS execution_traces (
			trace_id VARCHAR(255) PRIMARY KEY,
			workflow_id VARCHAR(255) NOT NULL,
			execution_id VARCHAR(255) NOT NULL,
			root_span_id VARCHAR(255) NOT NULL,
			start_time BIGINT NOT NULL,
			end_time BIGINT NULL,
			duration BIGINT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'running',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_workflow_id (workflow_id),
			INDEX idx_execution_id (execution_id),
			INDEX idx_status (status),
			INDEX idx_start_time (start_time)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		// 追踪跨度表
		`CREATE TABLE IF NOT EXISTS trace_spans (
			span_id VARCHAR(255) PRIMARY KEY,
			trace_id VARCHAR(255) NOT NULL,
			parent_span_id VARCHAR(255) NULL,
			name VARCHAR(255) NOT NULL,
			start_time BIGINT NOT NULL,
			end_time BIGINT NULL,
			duration BIGINT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'running',
			attributes JSON,
			events JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_trace_id (trace_id),
			INDEX idx_parent_span_id (parent_span_id),
			INDEX idx_name (name),
			INDEX idx_status (status),
			INDEX idx_start_time (start_time),
			FOREIGN KEY (trace_id) REFERENCES execution_traces(trace_id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %s, error: %w", query, err)
		}
	}

	return nil
}

// SaveExecution 保存执行记录
func (s *mysqlExecutionStorage) SaveExecution(result *types.ExecutionResult) error {
	inputJSON, err := json.Marshal(result.Input)
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}

	outputJSON, err := json.Marshal(result.Output)
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	query := `INSERT INTO executions (execution_id, workflow_id, status, input, output, error, start_time, end_time, duration, retry_count)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			  ON DUPLICATE KEY UPDATE 
			  status = VALUES(status), output = VALUES(output), error = VALUES(error),
			  end_time = VALUES(end_time), duration = VALUES(duration), retry_count = VALUES(retry_count)`

	_, err = s.db.Exec(query, result.ExecutionID, result.WorkflowID, result.Status,
		string(inputJSON), string(outputJSON), result.Error,
		result.StartTime, result.EndTime, result.Duration.Nanoseconds(), result.RetryCount)

	if err != nil {
		return fmt.Errorf("failed to save execution: %w", err)
	}

	return nil
}

// GetExecution 获取执行记录
func (s *mysqlExecutionStorage) GetExecution(executionID string) (*types.ExecutionResult, error) {
	query := `SELECT execution_id, workflow_id, status, input, output, error, start_time, end_time, duration, retry_count, created_at, updated_at
			  FROM executions WHERE execution_id = ?`

	row := s.db.QueryRow(query, executionID)

	var result types.ExecutionResult
	var inputJSON, outputJSON string
	var duration int64

	err := row.Scan(&result.ExecutionID, &result.WorkflowID, &result.Status,
		&inputJSON, &outputJSON, &result.Error,
		&result.StartTime, &result.EndTime, &duration, &result.RetryCount,
		&result.CreatedAt, &result.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("execution not found: %s", executionID)
		}
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	result.Duration = time.Duration(duration)

	// 反序列化JSON字段
	if err := json.Unmarshal([]byte(inputJSON), &result.Input); err != nil {
		return nil, fmt.Errorf("failed to unmarshal input: %w", err)
	}

	if err := json.Unmarshal([]byte(outputJSON), &result.Output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output: %w", err)
	}

	// 获取任务执行结果
	taskResults, err := s.ListTaskExecutions(executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task executions: %w", err)
	}

	result.TaskResults = make(map[string]*types.TaskExecutionResult)
	for _, taskResult := range taskResults {
		result.TaskResults[taskResult.TaskID] = taskResult
	}

	return &result, nil
}

// UpdateExecution 更新执行记录
func (s *mysqlExecutionStorage) UpdateExecution(result *types.ExecutionResult) error {
	return s.SaveExecution(result)
}

// ListExecutions 列出执行记录
func (s *mysqlExecutionStorage) ListExecutions(workflowID string, offset, limit int) ([]*types.ExecutionResult, error) {
	query := `SELECT execution_id, workflow_id, status, input, output, error, start_time, end_time, duration, retry_count, created_at, updated_at
			  FROM executions WHERE workflow_id = ? ORDER BY start_time DESC LIMIT ? OFFSET ?`

	rows, err := s.db.Query(query, workflowID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var results []*types.ExecutionResult
	for rows.Next() {
		var result types.ExecutionResult
		var inputJSON, outputJSON string
		var duration int64

		err := rows.Scan(&result.ExecutionID, &result.WorkflowID, &result.Status,
			&inputJSON, &outputJSON, &result.Error,
			&result.StartTime, &result.EndTime, &duration, &result.RetryCount,
			&result.CreatedAt, &result.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		result.Duration = time.Duration(duration)

		// 反序列化JSON字段
		if err := json.Unmarshal([]byte(inputJSON), &result.Input); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input: %w", err)
		}

		if err := json.Unmarshal([]byte(outputJSON), &result.Output); err != nil {
			return nil, fmt.Errorf("failed to unmarshal output: %w", err)
		}

		results = append(results, &result)
	}

	return results, nil
}

// SaveTaskExecution 保存任务执行记录
func (s *mysqlExecutionStorage) SaveTaskExecution(executionID string, result *types.TaskExecutionResult) error {
	inputJSON, err := json.Marshal(result.Input)
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}

	outputJSON, err := json.Marshal(result.Output)
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	query := `INSERT INTO task_executions (execution_id, task_id, status, input, output, error, start_time, end_time, duration, retry_count)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			  ON DUPLICATE KEY UPDATE 
			  status = VALUES(status), output = VALUES(output), error = VALUES(error),
			  end_time = VALUES(end_time), duration = VALUES(duration), retry_count = VALUES(retry_count)`

	_, err = s.db.Exec(query, executionID, result.TaskID, result.Status,
		string(inputJSON), string(outputJSON), result.Error,
		result.StartTime, result.EndTime, result.Duration.Nanoseconds(), result.RetryCount)

	if err != nil {
		return fmt.Errorf("failed to save task execution: %w", err)
	}

	return nil
}

// GetTaskExecution 获取任务执行记录
func (s *mysqlExecutionStorage) GetTaskExecution(executionID, taskID string) (*types.TaskExecutionResult, error) {
	query := `SELECT execution_id, task_id, status, input, output, error, start_time, end_time, duration, retry_count, created_at, updated_at
			  FROM task_executions WHERE execution_id = ? AND task_id = ?`

	row := s.db.QueryRow(query, executionID, taskID)

	var result types.TaskExecutionResult
	var inputJSON, outputJSON string
	var duration int64
	var createdAt, updatedAt time.Time

	err := row.Scan(&result.TaskID, &result.TaskID, &result.Status,
		&inputJSON, &outputJSON, &result.Error,
		&result.StartTime, &result.EndTime, &duration, &result.RetryCount,
		&createdAt, &updatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task execution not found: %s/%s", executionID, taskID)
		}
		return nil, fmt.Errorf("failed to get task execution: %w", err)
	}

	result.Duration = time.Duration(duration)

	// 反序列化JSON字段
	if err := json.Unmarshal([]byte(inputJSON), &result.Input); err != nil {
		return nil, fmt.Errorf("failed to unmarshal input: %w", err)
	}

	if err := json.Unmarshal([]byte(outputJSON), &result.Output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output: %w", err)
	}

	return &result, nil
}

// ListTaskExecutions 列出任务执行记录
func (s *mysqlExecutionStorage) ListTaskExecutions(executionID string) ([]*types.TaskExecutionResult, error) {
	query := `SELECT execution_id, task_id, status, input, output, error, start_time, end_time, duration, retry_count, created_at, updated_at
			  FROM task_executions WHERE execution_id = ? ORDER BY start_time ASC`

	rows, err := s.db.Query(query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list task executions: %w", err)
	}
	defer rows.Close()

	var results []*types.TaskExecutionResult
	for rows.Next() {
		var result types.TaskExecutionResult
		var inputJSON, outputJSON string
		var duration int64
		var createdAt, updatedAt time.Time

		err := rows.Scan(&result.TaskID, &result.TaskID, &result.Status,
			&inputJSON, &outputJSON, &result.Error,
			&result.StartTime, &result.EndTime, &duration, &result.RetryCount,
			&createdAt, &updatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan task execution: %w", err)
		}

		result.Duration = time.Duration(duration)

		// 反序列化JSON字段
		if err := json.Unmarshal([]byte(inputJSON), &result.Input); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input: %w", err)
		}

		if err := json.Unmarshal([]byte(outputJSON), &result.Output); err != nil {
			return nil, fmt.Errorf("failed to unmarshal output: %w", err)
		}

		results = append(results, &result)
	}

	return results, nil
}

// SaveRetryInfo 保存重试信息
func (s *mysqlExecutionStorage) SaveRetryInfo(info *types.RetryInfo) error {
	inputJSON, err := json.Marshal(info.Input)
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}

	manualRetriesJSON, err := json.Marshal(info.ManualRetries)
	if err != nil {
		return fmt.Errorf("failed to marshal manual retries: %w", err)
	}

	query := `INSERT INTO retry_info (execution_id, workflow_id, input, failure_reason, failed_at, retry_count, last_retry_at, status, manual_retries)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			  ON DUPLICATE KEY UPDATE 
			  retry_count = VALUES(retry_count), last_retry_at = VALUES(last_retry_at),
			  status = VALUES(status), manual_retries = VALUES(manual_retries)`

	_, err = s.db.Exec(query, info.ExecutionID, info.WorkflowID, string(inputJSON),
		info.FailureReason, info.FailedAt, info.RetryCount, info.LastRetryAt,
		info.Status, string(manualRetriesJSON))

	if err != nil {
		return fmt.Errorf("failed to save retry info: %w", err)
	}

	return nil
}

// GetRetryInfo 获取重试信息
func (s *mysqlExecutionStorage) GetRetryInfo(executionID string) (*types.RetryInfo, error) {
	query := `SELECT execution_id, workflow_id, input, failure_reason, failed_at, retry_count, last_retry_at, status, manual_retries, created_at, updated_at
			  FROM retry_info WHERE execution_id = ?`

	row := s.db.QueryRow(query, executionID)

	var info types.RetryInfo
	var inputJSON, manualRetriesJSON string
	var createdAt, updatedAt time.Time

	err := row.Scan(&info.ExecutionID, &info.WorkflowID, &inputJSON,
		&info.FailureReason, &info.FailedAt, &info.RetryCount, &info.LastRetryAt,
		&info.Status, &manualRetriesJSON, &createdAt, &updatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("retry info not found: %s", executionID)
		}
		return nil, fmt.Errorf("failed to get retry info: %w", err)
	}

	// 反序列化JSON字段
	if err := json.Unmarshal([]byte(inputJSON), &info.Input); err != nil {
		return nil, fmt.Errorf("failed to unmarshal input: %w", err)
	}

	if err := json.Unmarshal([]byte(manualRetriesJSON), &info.ManualRetries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manual retries: %w", err)
	}

	return &info, nil
}

// ListFailedExecutions 列出失败的执行
func (s *mysqlExecutionStorage) ListFailedExecutions() ([]*types.RetryInfo, error) {
	query := `SELECT execution_id, workflow_id, input, failure_reason, failed_at, retry_count, last_retry_at, status, manual_retries, created_at, updated_at
			  FROM retry_info WHERE status IN ('pending', 'exhausted') ORDER BY failed_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list failed executions: %w", err)
	}
	defer rows.Close()

	var infos []*types.RetryInfo
	for rows.Next() {
		var info types.RetryInfo
		var inputJSON, manualRetriesJSON string
		var createdAt, updatedAt time.Time

		err := rows.Scan(&info.ExecutionID, &info.WorkflowID, &inputJSON,
			&info.FailureReason, &info.FailedAt, &info.RetryCount, &info.LastRetryAt,
			&info.Status, &manualRetriesJSON, &createdAt, &updatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan retry info: %w", err)
		}

		// 反序列化JSON字段
		if err := json.Unmarshal([]byte(inputJSON), &info.Input); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input: %w", err)
		}

		if err := json.Unmarshal([]byte(manualRetriesJSON), &info.ManualRetries); err != nil {
			return nil, fmt.Errorf("failed to unmarshal manual retries: %w", err)
		}

		infos = append(infos, &info)
	}

	return infos, nil
}

// UpdateRetryInfo 更新重试信息
func (s *mysqlExecutionStorage) UpdateRetryInfo(info *types.RetryInfo) error {
	return s.SaveRetryInfo(info)
}

// DeleteRetryInfo 删除重试信息
func (s *mysqlExecutionStorage) DeleteRetryInfo(executionID string) error {
	query := `DELETE FROM retry_info WHERE execution_id = ?`
	result, err := s.db.Exec(query, executionID)
	if err != nil {
		return fmt.Errorf("failed to delete retry info: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("retry info not found: %s", executionID)
	}

	return nil
}

// SaveTrace 保存执行追踪记录
func (s *mysqlExecutionStorage) SaveTrace(trace *types.ExecutionTrace) error {
	query := `INSERT INTO execution_traces (trace_id, workflow_id, execution_id, root_span_id, start_time, end_time, duration, status)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			  ON DUPLICATE KEY UPDATE 
			  end_time = VALUES(end_time), duration = VALUES(duration), status = VALUES(status)`

	_, err := s.db.Exec(query, trace.TraceID, trace.WorkflowID, trace.ExecutionID, 
		trace.RootSpanID, trace.StartTime, trace.EndTime, trace.Duration, trace.Status)

	if err != nil {
		return fmt.Errorf("failed to save trace: %w", err)
	}

	return nil
}

// GetTrace 获取执行追踪记录
func (s *mysqlExecutionStorage) GetTrace(traceID string) (*types.ExecutionTrace, error) {
	query := `SELECT trace_id, workflow_id, execution_id, root_span_id, start_time, end_time, duration, status, created_at, updated_at
			  FROM execution_traces WHERE trace_id = ?`

	row := s.db.QueryRow(query, traceID)

	var trace types.ExecutionTrace
	err := row.Scan(&trace.TraceID, &trace.WorkflowID, &trace.ExecutionID,
		&trace.RootSpanID, &trace.StartTime, &trace.EndTime, &trace.Duration,
		&trace.Status, &trace.CreatedAt, &trace.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("trace not found: %s", traceID)
		}
		return nil, fmt.Errorf("failed to get trace: %w", err)
	}

	return &trace, nil
}

// SaveSpan 保存追踪跨度
func (s *mysqlExecutionStorage) SaveSpan(span *types.TraceSpan) error {
	attributesJSON, err := json.Marshal(span.Attributes)
	if err != nil {
		return fmt.Errorf("failed to marshal attributes: %w", err)
	}

	eventsJSON, err := json.Marshal(span.Events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	query := `INSERT INTO trace_spans (span_id, trace_id, parent_span_id, name, start_time, end_time, duration, status, attributes, events)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			  ON DUPLICATE KEY UPDATE 
			  end_time = VALUES(end_time), duration = VALUES(duration), status = VALUES(status),
			  attributes = VALUES(attributes), events = VALUES(events)`

	_, err = s.db.Exec(query, span.SpanID, span.TraceID, span.ParentSpanID, span.Name,
		span.StartTime, span.EndTime, span.Duration, span.Status,
		string(attributesJSON), string(eventsJSON))

	if err != nil {
		return fmt.Errorf("failed to save span: %w", err)
	}

	return nil
}

// GetSpan 获取追踪跨度
func (s *mysqlExecutionStorage) GetSpan(spanID string) (*types.TraceSpan, error) {
	query := `SELECT span_id, trace_id, parent_span_id, name, start_time, end_time, duration, status, attributes, events, created_at, updated_at
			  FROM trace_spans WHERE span_id = ?`

	row := s.db.QueryRow(query, spanID)

	var span types.TraceSpan
	var attributesJSON, eventsJSON string
	err := row.Scan(&span.SpanID, &span.TraceID, &span.ParentSpanID, &span.Name,
		&span.StartTime, &span.EndTime, &span.Duration, &span.Status,
		&attributesJSON, &eventsJSON, &span.CreatedAt, &span.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("span not found: %s", spanID)
		}
		return nil, fmt.Errorf("failed to get span: %w", err)
	}

	// 反序列化JSON字段
	if err := json.Unmarshal([]byte(attributesJSON), &span.Attributes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	if err := json.Unmarshal([]byte(eventsJSON), &span.Events); err != nil {
		return nil, fmt.Errorf("failed to unmarshal events: %w", err)
	}

	return &span, nil
}

// GetTraceSpans 获取追踪的所有跨度
func (s *mysqlExecutionStorage) GetTraceSpans(traceID string) ([]*types.TraceSpan, error) {
	query := `SELECT span_id, trace_id, parent_span_id, name, start_time, end_time, duration, status, attributes, events, created_at, updated_at
			  FROM trace_spans WHERE trace_id = ? ORDER BY start_time ASC`

	rows, err := s.db.Query(query, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trace spans: %w", err)
	}
	defer rows.Close()

	var spans []*types.TraceSpan
	for rows.Next() {
		var span types.TraceSpan
		var attributesJSON, eventsJSON string
		err := rows.Scan(&span.SpanID, &span.TraceID, &span.ParentSpanID, &span.Name,
			&span.StartTime, &span.EndTime, &span.Duration, &span.Status,
			&attributesJSON, &eventsJSON, &span.CreatedAt, &span.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan span: %w", err)
		}

		// 反序列化JSON字段
		if err := json.Unmarshal([]byte(attributesJSON), &span.Attributes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
		}

		if err := json.Unmarshal([]byte(eventsJSON), &span.Events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal events: %w", err)
		}

		spans = append(spans, &span)
	}

	return spans, nil
}

// ListTraces 列出执行追踪记录
func (s *mysqlExecutionStorage) ListTraces(workflowID string, offset, limit int) ([]*types.ExecutionTrace, error) {
	query := `SELECT trace_id, workflow_id, execution_id, root_span_id, start_time, end_time, duration, status, created_at, updated_at
			  FROM execution_traces WHERE workflow_id = ? ORDER BY start_time DESC LIMIT ? OFFSET ?`

	rows, err := s.db.Query(query, workflowID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list traces: %w", err)
	}
	defer rows.Close()

	var traces []*types.ExecutionTrace
	for rows.Next() {
		var trace types.ExecutionTrace
		err := rows.Scan(&trace.TraceID, &trace.WorkflowID, &trace.ExecutionID,
			&trace.RootSpanID, &trace.StartTime, &trace.EndTime, &trace.Duration,
			&trace.Status, &trace.CreatedAt, &trace.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan trace: %w", err)
		}

		traces = append(traces, &trace)
	}

	return traces, nil
}

// Close 关闭存储连接
func (s *mysqlExecutionStorage) Close() error {
	return s.db.Close()
}

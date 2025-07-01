package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/XXueTu/graph_task/engine"
)

// ExecutionStorage 执行数据存储（仅存储执行记录，不存储工作流定义）
type ExecutionStorage interface {
	// 执行记录管理
	SaveExecution(result *engine.ExecutionResult) error
	GetExecution(executionID string) (*engine.ExecutionResult, error)
	UpdateExecution(result *engine.ExecutionResult) error
	ListExecutions(workflowID string, offset, limit int) ([]*engine.ExecutionResult, error)

	// 任务执行记录管理
	SaveTaskExecution(executionID string, result *engine.TaskExecutionResult) error
	GetTaskExecution(executionID, taskID string) (*engine.TaskExecutionResult, error)
	ListTaskExecutions(executionID string) ([]*engine.TaskExecutionResult, error)

	// 重试记录管理
	SaveRetryInfo(info *engine.RetryInfo) error
	GetRetryInfo(executionID string) (*engine.RetryInfo, error)
	ListFailedExecutions() ([]*engine.RetryInfo, error)
	UpdateRetryInfo(info *engine.RetryInfo) error
	DeleteRetryInfo(executionID string) error

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
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %s, error: %w", query, err)
		}
	}

	return nil
}

// SaveExecution 保存执行记录
func (s *mysqlExecutionStorage) SaveExecution(result *engine.ExecutionResult) error {
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
func (s *mysqlExecutionStorage) GetExecution(executionID string) (*engine.ExecutionResult, error) {
	query := `SELECT execution_id, workflow_id, status, input, output, error, start_time, end_time, duration, retry_count, created_at, updated_at
			  FROM executions WHERE execution_id = ?`

	row := s.db.QueryRow(query, executionID)

	var result engine.ExecutionResult
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

	result.TaskResults = make(map[string]*engine.TaskExecutionResult)
	for _, taskResult := range taskResults {
		result.TaskResults[taskResult.TaskID] = taskResult
	}

	return &result, nil
}

// UpdateExecution 更新执行记录
func (s *mysqlExecutionStorage) UpdateExecution(result *engine.ExecutionResult) error {
	return s.SaveExecution(result)
}

// ListExecutions 列出执行记录
func (s *mysqlExecutionStorage) ListExecutions(workflowID string, offset, limit int) ([]*engine.ExecutionResult, error) {
	query := `SELECT execution_id, workflow_id, status, input, output, error, start_time, end_time, duration, retry_count, created_at, updated_at
			  FROM executions WHERE workflow_id = ? ORDER BY start_time DESC LIMIT ? OFFSET ?`

	rows, err := s.db.Query(query, workflowID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var results []*engine.ExecutionResult
	for rows.Next() {
		var result engine.ExecutionResult
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
func (s *mysqlExecutionStorage) SaveTaskExecution(executionID string, result *engine.TaskExecutionResult) error {
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
func (s *mysqlExecutionStorage) GetTaskExecution(executionID, taskID string) (*engine.TaskExecutionResult, error) {
	query := `SELECT execution_id, task_id, status, input, output, error, start_time, end_time, duration, retry_count, created_at, updated_at
			  FROM task_executions WHERE execution_id = ? AND task_id = ?`

	row := s.db.QueryRow(query, executionID, taskID)

	var result engine.TaskExecutionResult
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
func (s *mysqlExecutionStorage) ListTaskExecutions(executionID string) ([]*engine.TaskExecutionResult, error) {
	query := `SELECT execution_id, task_id, status, input, output, error, start_time, end_time, duration, retry_count, created_at, updated_at
			  FROM task_executions WHERE execution_id = ? ORDER BY start_time ASC`

	rows, err := s.db.Query(query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list task executions: %w", err)
	}
	defer rows.Close()

	var results []*engine.TaskExecutionResult
	for rows.Next() {
		var result engine.TaskExecutionResult
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
func (s *mysqlExecutionStorage) SaveRetryInfo(info *engine.RetryInfo) error {
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
func (s *mysqlExecutionStorage) GetRetryInfo(executionID string) (*engine.RetryInfo, error) {
	query := `SELECT execution_id, workflow_id, input, failure_reason, failed_at, retry_count, last_retry_at, status, manual_retries, created_at, updated_at
			  FROM retry_info WHERE execution_id = ?`

	row := s.db.QueryRow(query, executionID)

	var info engine.RetryInfo
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
func (s *mysqlExecutionStorage) ListFailedExecutions() ([]*engine.RetryInfo, error) {
	query := `SELECT execution_id, workflow_id, input, failure_reason, failed_at, retry_count, last_retry_at, status, manual_retries, created_at, updated_at
			  FROM retry_info WHERE status IN ('pending', 'exhausted') ORDER BY failed_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list failed executions: %w", err)
	}
	defer rows.Close()

	var infos []*engine.RetryInfo
	for rows.Next() {
		var info engine.RetryInfo
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
func (s *mysqlExecutionStorage) UpdateRetryInfo(info *engine.RetryInfo) error {
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

// Close 关闭存储连接
func (s *mysqlExecutionStorage) Close() error {
	return s.db.Close()
}

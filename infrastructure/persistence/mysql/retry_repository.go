package mysql

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/XXueTu/graph_task/domain/retry"
)

// retryRepository MySQL重试仓储实现
type retryRepository struct {
	db *sql.DB
}

// NewRetryRepository 创建MySQL重试仓储
func NewRetryRepository(dsn string) (retry.Repository, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	repo := &retryRepository{db: db}

	// 初始化表结构
	if err := repo.initTables(); err != nil {
		return nil, err
	}

	return repo, nil
}

// initTables 初始化数据库表
func (r *retryRepository) initTables() error {
	query := `CREATE TABLE IF NOT EXISTS retry_info (
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
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

	_, err := r.db.Exec(query)
	return err
}

// SaveRetryInfo 保存重试信息
func (r *retryRepository) SaveRetryInfo(info *retry.Info) error {
	inputJSON, err := json.Marshal(info.Input())
	if err != nil {
		return err
	}

	manualRetriesJSON, err := json.Marshal(info.ManualRetries())
	if err != nil {
		return err
	}

	query := `INSERT INTO retry_info (execution_id, workflow_id, input, failure_reason, failed_at, retry_count, last_retry_at, status, manual_retries)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			  ON DUPLICATE KEY UPDATE 
			  retry_count = VALUES(retry_count), last_retry_at = VALUES(last_retry_at),
			  status = VALUES(status), manual_retries = VALUES(manual_retries)`

	_, err = r.db.Exec(query, info.ExecutionID(), info.WorkflowID(), string(inputJSON),
		info.FailureReason(), info.FailedAt(), info.RetryCount(), info.LastRetryAt(),
		string(info.Status()), string(manualRetriesJSON))

	return err
}

// FindRetryInfo 根据执行ID查找重试信息
func (r *retryRepository) FindRetryInfo(executionID string) (*retry.Info, error) {
	query := `SELECT execution_id, workflow_id, input, failure_reason, failed_at, retry_count, last_retry_at, status, manual_retries
			  FROM retry_info WHERE execution_id = ?`

	row := r.db.QueryRow(query, executionID)

	var inputJSON, manualRetriesJSON string
	var workflowID, failureReason, status string
	var failedAt time.Time
	var lastRetryAt *time.Time
	var retryCount int

	err := row.Scan(&executionID, &workflowID, &inputJSON, &failureReason, &failedAt, &retryCount, &lastRetryAt, &status, &manualRetriesJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewMySQLError("retry info not found: " + executionID)
		}
		return nil, err
	}

	// 反序列化JSON
	var input map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return nil, err
	}

	var manualRetries []*retry.ManualRetryRecord
	if err := json.Unmarshal([]byte(manualRetriesJSON), &manualRetries); err != nil {
		return nil, err
	}

	// 创建重试信息
	info := retry.NewInfo(executionID, workflowID, input, failureReason)
	
	// 设置状态
	switch retry.Status(status) {
	case retry.StatusPending:
		// 默认状态
	case retry.StatusExhausted:
		info.MarkExhausted()
	case retry.StatusAbandoned:
		info.MarkAbandoned()
	case retry.StatusRecovered:
		info.MarkRecovered()
	}

	// 添加手动重试记录
	for _, record := range manualRetries {
		info.AddManualRetry(record)
	}

	return info, nil
}

// ListFailedExecutions 列出失败的执行
func (r *retryRepository) ListFailedExecutions() ([]*retry.Info, error) {
	query := `SELECT execution_id FROM retry_info WHERE status IN ('pending', 'exhausted') ORDER BY failed_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var infos []*retry.Info
	for rows.Next() {
		var executionID string
		if err := rows.Scan(&executionID); err != nil {
			return nil, err
		}

		info, err := r.FindRetryInfo(executionID)
		if err != nil {
			return nil, err
		}

		infos = append(infos, info)
	}

	return infos, nil
}

// UpdateRetryInfo 更新重试信息
func (r *retryRepository) UpdateRetryInfo(info *retry.Info) error {
	return r.SaveRetryInfo(info)
}

// DeleteRetryInfo 删除重试信息
func (r *retryRepository) DeleteRetryInfo(executionID string) error {
	query := `DELETE FROM retry_info WHERE execution_id = ?`
	result, err := r.db.Exec(query, executionID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return NewMySQLError("retry info not found: " + executionID)
	}

	return nil
}

// GetStatistics 获取重试统计
func (r *retryRepository) GetStatistics() (*retry.Statistics, error) {
	query := `SELECT 
		COUNT(*) as total,
		SUM(CASE WHEN status = 'recovered' THEN 1 ELSE 0 END) as recovered,
		SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
		SUM(CASE WHEN status = 'exhausted' THEN 1 ELSE 0 END) as exhausted,
		SUM(CASE WHEN status = 'abandoned' THEN 1 ELSE 0 END) as abandoned
		FROM retry_info`

	row := r.db.QueryRow(query)

	var total, recovered, pending, exhausted, abandoned int
	err := row.Scan(&total, &recovered, &pending, &exhausted, &abandoned)
	if err != nil {
		return nil, err
	}

	return retry.NewStatistics(total, recovered, pending, exhausted, abandoned), nil
}
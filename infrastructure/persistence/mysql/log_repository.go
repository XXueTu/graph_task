package mysql

import (
	"database/sql"
	"encoding/json"

	"github.com/XXueTu/graph_task/domain/logger"
)

// logRepository MySQL日志仓储实现
type logRepository struct {
	db *sql.DB
}

// NewLogRepository 创建MySQL日志仓储
func NewLogRepository(dsn string) (logger.LogRepository, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	repo := &logRepository{db: db}

	// 初始化表结构
	if err := repo.initTables(); err != nil {
		return nil, err
	}

	return repo, nil
}

// initTables 初始化数据库表
func (r *logRepository) initTables() error {
	query := `CREATE TABLE IF NOT EXISTS execution_logs (
		id VARCHAR(255) PRIMARY KEY,
		execution_id VARCHAR(255) NOT NULL,
		task_id VARCHAR(255) NOT NULL DEFAULT '',
		level VARCHAR(10) NOT NULL,
		message TEXT NOT NULL,
		attributes JSON,
		timestamp TIMESTAMP(6) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_execution_id (execution_id),
		INDEX idx_task_id (task_id),
		INDEX idx_level (level),
		INDEX idx_timestamp (timestamp)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

	_, err := r.db.Exec(query)
	return err
}

// SaveLogs 批量保存日志
func (r *logRepository) SaveLogs(logs []*logger.LogEntry) error {
	if len(logs) == 0 {
		return nil
	}

	// 构建批量插入SQL
	query := `INSERT INTO execution_logs (id, execution_id, task_id, level, message, attributes, timestamp) VALUES `
	values := make([]interface{}, 0, len(logs)*7)

	for i, log := range logs {
		if i > 0 {
			query += ", "
		}
		query += "(?, ?, ?, ?, ?, ?, ?)"

		attributesJSON, err := json.Marshal(log.Attributes())
		if err != nil {
			return err
		}

		values = append(values, log.ID(), log.ExecutionID(), log.TaskID(),
			string(log.Level()), log.Message(), string(attributesJSON), log.Timestamp())
	}

	_, err := r.db.Exec(query, values...)
	return err
}

// GetLogs 获取执行日志
func (r *logRepository) GetLogs(executionID string, limit, offset int) ([]*logger.LogEntry, error) {
	query := `SELECT id, execution_id, task_id, level, message, attributes, timestamp 
			  FROM execution_logs WHERE execution_id = ? 
			  ORDER BY timestamp DESC LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, executionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*logger.LogEntry
	for rows.Next() {
		var id, execID, taskID, level, message, attributesJSON string
		var timestamp string

		err := rows.Scan(&id, &execID, &taskID, &level, &message, &attributesJSON, &timestamp)
		if err != nil {
			return nil, err
		}

		// 反序列化属性
		var attributes map[string]interface{}
		if err := json.Unmarshal([]byte(attributesJSON), &attributes); err != nil {
			return nil, err
		}

		// 重建日志条目
		log := logger.NewLogEntry(execID, taskID, logger.LogLevel(level), message, attributes)
		logs = append(logs, log)
	}

	return logs, nil
}

// GetTaskLogs 获取任务日志
func (r *logRepository) GetTaskLogs(executionID, taskID string, limit, offset int) ([]*logger.LogEntry, error) {
	query := `SELECT id, execution_id, task_id, level, message, attributes, timestamp 
			  FROM execution_logs WHERE execution_id = ? AND task_id = ? 
			  ORDER BY timestamp DESC LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, executionID, taskID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*logger.LogEntry
	for rows.Next() {
		var id, execID, tID, level, message, attributesJSON string
		var timestamp string

		err := rows.Scan(&id, &execID, &tID, &level, &message, &attributesJSON, &timestamp)
		if err != nil {
			return nil, err
		}

		// 反序列化属性
		var attributes map[string]interface{}
		if err := json.Unmarshal([]byte(attributesJSON), &attributes); err != nil {
			return nil, err
		}

		// 重建日志条目
		log := logger.NewLogEntry(execID, tID, logger.LogLevel(level), message, attributes)
		logs = append(logs, log)
	}

	return logs, nil
}

// DeleteLogs 删除日志
func (r *logRepository) DeleteLogs(executionID string) error {
	query := `DELETE FROM execution_logs WHERE execution_id = ?`
	_, err := r.db.Exec(query, executionID)
	return err
}

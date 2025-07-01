package mysql

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/XXueTu/graph_task/domain/execution"
	"github.com/XXueTu/graph_task/domain/workflow"
)

// executionRepository MySQL执行仓储实现
type executionRepository struct {
	db *sql.DB
}

// NewExecutionRepository 创建MySQL执行仓储
func NewExecutionRepository(dsn string) (execution.Repository, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	repo := &executionRepository{db: db}

	// 初始化表结构
	if err := repo.initTables(); err != nil {
		return nil, err
	}

	return repo, nil
}

// initTables 初始化数据库表
func (r *executionRepository) initTables() error {
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
		`CREATE TABLE IF NOT EXISTS task_results (
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
			INDEX idx_execution_id (execution_id),
			INDEX idx_task_id (task_id),
			INDEX idx_status (status),
			UNIQUE KEY uk_execution_task (execution_id, task_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

// SaveExecution 保存执行记录
func (r *executionRepository) SaveExecution(exec *execution.Execution) error {
	inputJSON, err := json.Marshal(exec.Input())
	if err != nil {
		return err
	}

	outputJSON, err := json.Marshal(exec.Output())
	if err != nil {
		return err
	}

	query := `INSERT INTO executions (execution_id, workflow_id, status, input, output, error, start_time, end_time, duration, retry_count)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			  ON DUPLICATE KEY UPDATE 
			  status = VALUES(status), output = VALUES(output), error = VALUES(error),
			  end_time = VALUES(end_time), duration = VALUES(duration), retry_count = VALUES(retry_count)`

	_, err = r.db.Exec(query, exec.ID(), exec.WorkflowID(), int(exec.Status()),
		string(inputJSON), string(outputJSON), exec.Error(),
		exec.StartTime(), exec.EndTime(), exec.Duration().Milliseconds(), exec.RetryCount())

	if err != nil {
		return err
	}

	// 保存任务结果
	for _, taskResult := range exec.TaskResults() {
		if err := r.saveTaskResult(exec.ID(), taskResult); err != nil {
			return err
		}
	}

	return nil
}

// saveTaskResult 保存任务结果
func (r *executionRepository) saveTaskResult(executionID string, result *execution.TaskResult) error {
	inputJSON, err := json.Marshal(result.Input())
	if err != nil {
		return err
	}

	outputJSON, err := json.Marshal(result.Output())
	if err != nil {
		return err
	}

	query := `INSERT INTO task_results (execution_id, task_id, status, input, output, error, start_time, end_time, duration, retry_count)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			  ON DUPLICATE KEY UPDATE 
			  status = VALUES(status), output = VALUES(output), error = VALUES(error),
			  end_time = VALUES(end_time), duration = VALUES(duration), retry_count = VALUES(retry_count)`

	_, err = r.db.Exec(query, executionID, result.TaskID(), int(result.Status()),
		string(inputJSON), string(outputJSON), result.Error(),
		result.StartTime(), result.EndTime(), result.Duration().Milliseconds(), result.RetryCount())

	return err
}

// FindExecutionByID 根据ID查找执行
func (r *executionRepository) FindExecutionByID(id string) (*execution.Execution, error) {
	query := `SELECT execution_id, workflow_id, status, input, output, error, start_time, end_time, duration, retry_count, created_at, updated_at
			  FROM executions WHERE execution_id = ?`

	row := r.db.QueryRow(query, id)

	var inputJSON, outputJSON string
	var duration int64
	var status int
	var startTime, endTime, createdAt, updatedAt time.Time
	var workflowID, errorMsg string
	var retryCount int
	var executionID string

	err := row.Scan(&executionID, &workflowID, &status, &inputJSON, &outputJSON, &errorMsg,
		&startTime, &endTime, &duration, &retryCount, &createdAt, &updatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewMySQLError("execution not found: " + id)
		}
		return nil, err
	}

	// 反序列化JSON
	var input, output map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(outputJSON), &output); err != nil {
		return nil, err
	}

	// 创建执行对象
	exec := execution.NewExecution(executionID, workflowID, input)
	exec.Start()

	if execution.Status(status) == execution.StatusSuccess {
		exec.Complete(output)
	} else if execution.Status(status) == execution.StatusFailed {
		exec.Fail(errorMsg)
	}

	// 加载任务结果
	taskResults, err := r.loadTaskResults(id)
	if err != nil {
		return nil, err
	}

	for _, taskResult := range taskResults {
		exec.SetTaskResult(taskResult.TaskID(), taskResult)
	}

	return exec, nil
}

// loadTaskResults 加载任务结果
func (r *executionRepository) loadTaskResults(executionID string) ([]*execution.TaskResult, error) {
	query := `SELECT task_id, status, input, output, error, start_time, end_time, duration, retry_count
			  FROM task_results WHERE execution_id = ? ORDER BY start_time ASC`

	rows, err := r.db.Query(query, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*execution.TaskResult
	for rows.Next() {
		var taskID, inputJSON, outputJSON, errorMsg string
		var status int
		var duration int64
		var startTime, endTime time.Time
		var retryCount int

		err := rows.Scan(&taskID, &status, &inputJSON, &outputJSON, &errorMsg,
			&startTime, &endTime, &duration, &retryCount)
		if err != nil {
			return nil, err
		}

		// 反序列化JSON
		var input, output map[string]interface{}
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(outputJSON), &output); err != nil {
			return nil, err
		}

		// 创建任务结果
		result := execution.NewTaskResult(taskID)
		result.Start(input)

		if workflow.TaskStatus(status) == workflow.TaskStatusSuccess {
			result.Complete(output)
		} else if workflow.TaskStatus(status) == workflow.TaskStatusFailed {
			result.Fail(errorMsg)
		}

		results = append(results, result)
	}

	return results, nil
}

// ListExecutions 列出执行记录
func (r *executionRepository) ListExecutions(workflowID string, limit, offset int) ([]*execution.Execution, error) {
	var query string
	var args []interface{}

	if workflowID == "" {
		query = `SELECT execution_id FROM executions ORDER BY start_time DESC LIMIT ? OFFSET ?`
		args = []interface{}{limit, offset}
	} else {
		query = `SELECT execution_id FROM executions WHERE workflow_id = ? ORDER BY start_time DESC LIMIT ? OFFSET ?`
		args = []interface{}{workflowID, limit, offset}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []*execution.Execution
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		exec, err := r.FindExecutionByID(id)
		if err != nil {
			return nil, err
		}

		executions = append(executions, exec)
	}

	return executions, nil
}

// UpdateExecution 更新执行记录
func (r *executionRepository) UpdateExecution(exec *execution.Execution) error {
	return r.SaveExecution(exec)
}

// DeleteExecution 删除执行记录
func (r *executionRepository) DeleteExecution(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 删除任务结果
	if _, err := tx.Exec("DELETE FROM task_results WHERE execution_id = ?", id); err != nil {
		return err
	}

	// 删除执行记录
	if _, err := tx.Exec("DELETE FROM executions WHERE execution_id = ?", id); err != nil {
		return err
	}

	return tx.Commit()
}

// ListRunningExecutions 列出正在运行的执行
func (r *executionRepository) ListRunningExecutions() ([]*execution.Execution, error) {
	query := `SELECT execution_id FROM executions WHERE status = ? ORDER BY start_time DESC`

	rows, err := r.db.Query(query, int(execution.StatusRunning))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []*execution.Execution
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		exec, err := r.FindExecutionByID(id)
		if err != nil {
			return nil, err
		}

		executions = append(executions, exec)
	}

	return executions, nil
}

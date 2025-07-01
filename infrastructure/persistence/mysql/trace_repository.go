package mysql

import (
	"database/sql"
	"encoding/json"

	"github.com/XXueTu/graph_task/domain/trace"
)

// traceRepository MySQL追踪仓储实现
type traceRepository struct {
	db *sql.DB
}

// NewTraceRepository 创建MySQL追踪仓储
func NewTraceRepository(dsn string) (trace.Repository, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	repo := &traceRepository{db: db}

	// 初始化表结构
	if err := repo.initTables(); err != nil {
		return nil, err
	}

	return repo, nil
}

// initTables 初始化数据库表
func (r *traceRepository) initTables() error {
	query := `CREATE TABLE IF NOT EXISTS trace_spans (
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
		INDEX idx_start_time (start_time)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

	_, err := r.db.Exec(query)
	return err
}

// SaveSpan 保存跨度
func (r *traceRepository) SaveSpan(span *trace.Span) error {
	attributesJSON, err := json.Marshal(span.Attributes())
	if err != nil {
		return err
	}

	eventsJSON, err := json.Marshal(span.Events())
	if err != nil {
		return err
	}

	query := `INSERT INTO trace_spans (span_id, trace_id, parent_span_id, name, start_time, end_time, duration, status, attributes, events)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			  ON DUPLICATE KEY UPDATE 
			  end_time = VALUES(end_time), duration = VALUES(duration), status = VALUES(status),
			  attributes = VALUES(attributes), events = VALUES(events)`

	_, err = r.db.Exec(query, span.SpanID(), span.TraceID(), span.ParentSpanID(), span.Name(),
		span.StartTime(), span.EndTime(), span.Duration(), string(span.Status()),
		string(attributesJSON), string(eventsJSON))

	return err
}

// FindSpan 根据ID查找跨度
func (r *traceRepository) FindSpan(spanID string) (*trace.Span, error) {
	query := `SELECT span_id, trace_id, parent_span_id, name, start_time, end_time, duration, status, attributes, events, created_at, updated_at
			  FROM trace_spans WHERE span_id = ?`

	row := r.db.QueryRow(query, spanID)

	var id, traceID, name, status string
	var parentSpanID *string
	var startTime int64
	var endTime, duration *int64
	var attributesJSON, eventsJSON string
	var createdAt, updatedAt string

	err := row.Scan(&id, &traceID, &parentSpanID, &name,
		&startTime, &endTime, &duration, &status,
		&attributesJSON, &eventsJSON, &createdAt, &updatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewMySQLError("span not found: " + spanID)
		}
		return nil, err
	}

	// 反序列化JSON字段
	var attributes map[string]interface{}
	if err := json.Unmarshal([]byte(attributesJSON), &attributes); err != nil {
		return nil, err
	}

	var events []*trace.Event
	if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
		return nil, err
	}

	// 重建跨度对象
	rebuiltSpan := trace.NewSpan(id, traceID, name)
	if parentSpanID != nil {
		rebuiltSpan.SetParentSpanID(*parentSpanID)
	}

	for key, value := range attributes {
		rebuiltSpan.SetAttribute(key, value)
	}

	for _, event := range events {
		rebuiltSpan.AddEvent(event)
	}

	return rebuiltSpan, nil
}

// FindTraceSpans 查找追踪的所有跨度
func (r *traceRepository) FindTraceSpans(traceID string) ([]*trace.Span, error) {
	query := `SELECT span_id FROM trace_spans WHERE trace_id = ? ORDER BY start_time ASC`

	rows, err := r.db.Query(query, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []*trace.Span
	for rows.Next() {
		var spanID string
		if err := rows.Scan(&spanID); err != nil {
			return nil, err
		}

		span, err := r.FindSpan(spanID)
		if err != nil {
			return nil, err
		}

		spans = append(spans, span)
	}

	return spans, nil
}

// DeleteSpan 删除跨度
func (r *traceRepository) DeleteSpan(spanID string) error {
	query := `DELETE FROM trace_spans WHERE span_id = ?`
	result, err := r.db.Exec(query, spanID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return NewMySQLError("span not found: " + spanID)
	}

	return nil
}

// DeleteTrace 删除追踪的所有跨度
func (r *traceRepository) DeleteTrace(traceID string) error {
	query := `DELETE FROM trace_spans WHERE trace_id = ?`
	_, err := r.db.Exec(query, traceID)
	return err
}
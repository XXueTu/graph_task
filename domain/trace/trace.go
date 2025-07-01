package trace

import (
	"time"
)

// SpanStatus 跨度状态
type SpanStatus string

const (
	SpanStatusRunning SpanStatus = "running"
	SpanStatusSuccess SpanStatus = "success"
	SpanStatusError   SpanStatus = "error"
)

// Event 追踪事件
type Event struct {
	timestamp  int64
	name       string
	attributes map[string]interface{}
}

// NewEvent 创建追踪事件
func NewEvent(name string, attributes map[string]interface{}) *Event {
	return &Event{
		timestamp:  time.Now().UnixNano(),
		name:       name,
		attributes: attributes,
	}
}

// Event getter methods
func (e *Event) Timestamp() int64 { return e.timestamp }
func (e *Event) Name() string { return e.name }
func (e *Event) Attributes() map[string]interface{} { return e.attributes }

// Span 追踪跨度
type Span struct {
	spanID       string
	traceID      string
	parentSpanID *string
	name         string
	startTime    int64
	endTime      *int64
	duration     *int64
	status       SpanStatus
	attributes   map[string]interface{}
	events       []*Event
	createdAt    time.Time
	updatedAt    time.Time
}

// NewSpan 创建新跨度
func NewSpan(spanID, traceID, name string) *Span {
	now := time.Now()
	return &Span{
		spanID:     spanID,
		traceID:    traceID,
		name:       name,
		startTime:  now.UnixNano(),
		status:     SpanStatusRunning,
		attributes: make(map[string]interface{}),
		events:     make([]*Event, 0),
		createdAt:  now,
		updatedAt:  now,
	}
}

// Span getter methods
func (s *Span) SpanID() string { return s.spanID }
func (s *Span) TraceID() string { return s.traceID }
func (s *Span) ParentSpanID() *string { return s.parentSpanID }
func (s *Span) Name() string { return s.name }
func (s *Span) StartTime() int64 { return s.startTime }
func (s *Span) EndTime() *int64 { return s.endTime }
func (s *Span) Duration() *int64 { return s.duration }
func (s *Span) Status() SpanStatus { return s.status }
func (s *Span) Attributes() map[string]interface{} { return s.attributes }
func (s *Span) Events() []*Event { return s.events }
func (s *Span) CreatedAt() time.Time { return s.createdAt }
func (s *Span) UpdatedAt() time.Time { return s.updatedAt }

// SetParentSpanID 设置父跨度ID
func (s *Span) SetParentSpanID(parentSpanID string) {
	s.parentSpanID = &parentSpanID
	s.updatedAt = time.Now()
}

// SetAttribute 设置属性
func (s *Span) SetAttribute(key string, value interface{}) {
	s.attributes[key] = value
	s.updatedAt = time.Now()
}

// AddEvent 添加事件
func (s *Span) AddEvent(event *Event) {
	s.events = append(s.events, event)
	s.updatedAt = time.Now()
}

// End 结束跨度
func (s *Span) End(status SpanStatus) {
	now := time.Now()
	endTime := now.UnixNano()
	duration := endTime - s.startTime
	
	s.endTime = &endTime
	s.duration = &duration
	s.status = status
	s.updatedAt = now
}

// IsRunning 是否正在运行
func (s *Span) IsRunning() bool {
	return s.status == SpanStatusRunning
}

// IsCompleted 是否已完成
func (s *Span) IsCompleted() bool {
	return s.status == SpanStatusSuccess || s.status == SpanStatusError
}

// Trace 追踪聚合根
type Trace struct {
	traceID   string
	spans     map[string]*Span
	createdAt time.Time
	updatedAt time.Time
}

// NewTrace 创建新追踪
func NewTrace(traceID string) *Trace {
	now := time.Now()
	return &Trace{
		traceID:   traceID,
		spans:     make(map[string]*Span),
		createdAt: now,
		updatedAt: now,
	}
}

// Trace getter methods
func (t *Trace) TraceID() string { return t.traceID }
func (t *Trace) Spans() map[string]*Span { return t.spans }
func (t *Trace) CreatedAt() time.Time { return t.createdAt }
func (t *Trace) UpdatedAt() time.Time { return t.updatedAt }

// AddSpan 添加跨度
func (t *Trace) AddSpan(span *Span) {
	t.spans[span.SpanID()] = span
	t.updatedAt = time.Now()
}

// GetSpan 获取跨度
func (t *Trace) GetSpan(spanID string) (*Span, bool) {
	span, exists := t.spans[spanID]
	return span, exists
}

// GetRootSpans 获取根跨度
func (t *Trace) GetRootSpans() []*Span {
	var rootSpans []*Span
	for _, span := range t.spans {
		if span.ParentSpanID() == nil {
			rootSpans = append(rootSpans, span)
		}
	}
	return rootSpans
}

// GetChildSpans 获取子跨度
func (t *Trace) GetChildSpans(parentSpanID string) []*Span {
	var childSpans []*Span
	for _, span := range t.spans {
		if span.ParentSpanID() != nil && *span.ParentSpanID() == parentSpanID {
			childSpans = append(childSpans, span)
		}
	}
	return childSpans
}
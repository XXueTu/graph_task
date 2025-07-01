package execution

import "time"

// Event 执行事件
type Event struct {
	id          string
	eventType   string
	executionID string
	workflowID  string
	taskID      string
	data        map[string]interface{}
	timestamp   time.Time
}

// NewExecutionEvent 创建执行事件
func NewExecutionEvent(eventType, executionID, workflowID, taskID string, data map[string]interface{}) *Event {
	return &Event{
		eventType:   eventType,
		executionID: executionID,
		workflowID:  workflowID,
		taskID:      taskID,
		data:        data,
		timestamp:   time.Now(),
	}
}

// Event getter methods
func (e *Event) ID() string { return e.id }
func (e *Event) Type() string { return e.eventType }
func (e *Event) ExecutionID() string { return e.executionID }
func (e *Event) WorkflowID() string { return e.workflowID }
func (e *Event) TaskID() string { return e.taskID }
func (e *Event) Data() map[string]interface{} { return e.data }
func (e *Event) Timestamp() time.Time { return e.timestamp }

// EventPublisher 事件发布器接口
type EventPublisher interface {
	// Publish 发布事件
	Publish(event *Event) error
}

// EventHandler 事件处理器
type EventHandler func(event *Event) error

// EventSubscriber 事件订阅器接口
type EventSubscriber interface {
	// Subscribe 订阅事件
	Subscribe(eventType string, handler EventHandler) error
	
	// Unsubscribe 取消订阅
	Unsubscribe(eventType string, handler EventHandler) error
}
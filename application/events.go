package application

import (
	"time"
)

// Event 应用层事件
type Event struct {
	id        string
	eventType string
	entityID  string
	data      map[string]interface{}
	timestamp time.Time
}

// NewEvent 创建事件
func NewEvent(eventType, entityID string, data map[string]interface{}) *Event {
	return &Event{
		eventType: eventType,
		entityID:  entityID,
		data:      data,
		timestamp: time.Now(),
	}
}

// Event getter methods
func (e *Event) ID() string { return e.id }
func (e *Event) Type() string { return e.eventType }
func (e *Event) EntityID() string { return e.entityID }
func (e *Event) Data() map[string]interface{} { return e.data }
func (e *Event) Timestamp() time.Time { return e.timestamp }

// NewWorkflowEvent 创建工作流事件
func NewWorkflowEvent(eventType, workflowID string, data map[string]interface{}) *Event {
	return NewEvent(eventType, workflowID, data)
}

// NewExecutionEvent 创建执行事件
func NewExecutionEvent(eventType, executionID string, data map[string]interface{}) *Event {
	return NewEvent(eventType, executionID, data)
}

// NewRetryEvent 创建重试事件
func NewRetryEvent(eventType, executionID, workflowID string, data map[string]interface{}) *Event {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["workflow_id"] = workflowID
	return NewEvent(eventType, executionID, data)
}

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
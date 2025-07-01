package event

import (
	"log"
	"sync"
)

// EventBus 事件总线接口
type EventBus interface {
	Publish(event *Event) error
	Subscribe(eventType string, handler EventHandler) error
}

// EventHandler 事件处理器
type EventHandler func(event *Event) error

// Event 事件
type Event struct {
	Type        string                 `json:"type"`
	WorkflowID  string                 `json:"workflow_id"`
	ExecutionID string                 `json:"execution_id"`
	TaskID      string                 `json:"task_id,omitempty"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   int64                  `json:"timestamp"`
}

// eventBus 事件总线实现
type eventBus struct {
	handlers map[string][]EventHandler
	mutex    sync.RWMutex
}

// NewEventBus 创建事件总线
func NewEventBus() EventBus {
	return &eventBus{
		handlers: make(map[string][]EventHandler),
	}
}

// Publish 发布事件
func (eb *eventBus) Publish(event *Event) error {
	eb.mutex.RLock()
	handlers, exists := eb.handlers[event.Type]
	eb.mutex.RUnlock()

	if !exists {
		return nil // 没有订阅者，直接返回
	}

	// 异步处理事件
	go func() {
		for _, handler := range handlers {
			if err := handler(event); err != nil {
				log.Printf("事件处理失败: %v", err)
			}
		}
	}()

	return nil
}

// Subscribe 订阅事件
func (eb *eventBus) Subscribe(eventType string, handler EventHandler) error {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	if eb.handlers[eventType] == nil {
		eb.handlers[eventType] = make([]EventHandler, 0)
	}

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	return nil
}

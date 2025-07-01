package eventbus

import (
	"log"
	"sync"

	"github.com/XXueTu/graph_task/application"
)

// EventBus 事件总线实现
type EventBus struct {
	handlers map[string][]application.EventHandler
	mutex    sync.RWMutex
}

// NewEventBus 创建事件总线
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]application.EventHandler),
	}
}

// Publish 发布事件
func (eb *EventBus) Publish(event *application.Event) error {
	eb.mutex.RLock()
	handlers, exists := eb.handlers[event.Type()]
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
func (eb *EventBus) Subscribe(eventType string, handler application.EventHandler) error {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	if eb.handlers[eventType] == nil {
		eb.handlers[eventType] = make([]application.EventHandler, 0)
	}

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	return nil
}

// Unsubscribe 取消订阅
func (eb *EventBus) Unsubscribe(eventType string, handler application.EventHandler) error {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	handlers := eb.handlers[eventType]
	if handlers == nil {
		return nil
	}

	// 移除处理器（简化实现，实际需要比较函数指针）
	eb.handlers[eventType] = make([]application.EventHandler, 0)
	return nil
}
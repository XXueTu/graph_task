package execution

import (
	"sync"
	"time"

	"github.com/XXueTu/graph_task/domain/trace"
)

// TracerService 执行追踪服务
type TracerService interface {
	// StartTrace 开始追踪
	StartTrace(executionID string) *trace.Trace
	
	// StartSpan 开始跨度
	StartSpan(traceID, spanID, name string, parentSpanID *string) *trace.Span
	
	// EndSpan 结束跨度
	EndSpan(spanID string, status trace.SpanStatus) error
	
	// AddSpanEvent 添加跨度事件
	AddSpanEvent(spanID, eventName string, attributes map[string]interface{}) error
	
	// SetSpanAttribute 设置跨度属性
	SetSpanAttribute(spanID, key string, value interface{}) error
	
	// FlushTraces 批量刷新追踪数据
	FlushTraces() error
	
	// GetTrace 获取追踪
	GetTrace(traceID string) (*trace.Trace, error)
	
	// Close 关闭追踪器
	Close() error
}

// tracerService 追踪服务实现
type tracerService struct {
	repository   trace.Repository
	traces       map[string]*trace.Trace
	spans        map[string]*trace.Span
	batchSize    int
	flushInterval time.Duration
	pendingSpans []*trace.Span
	mutex        sync.RWMutex
	stopCh       chan struct{}
}

// NewTracerService 创建追踪服务
func NewTracerService(repository trace.Repository, batchSize int, flushInterval time.Duration) TracerService {
	service := &tracerService{
		repository:    repository,
		traces:        make(map[string]*trace.Trace),
		spans:         make(map[string]*trace.Span),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		pendingSpans:  make([]*trace.Span, 0, batchSize),
		stopCh:        make(chan struct{}),
	}
	
	// 启动批量刷新协程
	go service.backgroundFlush()
	
	return service
}

// StartTrace 开始追踪
func (t *tracerService) StartTrace(executionID string) *trace.Trace {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	tr := trace.NewTrace(executionID)
	t.traces[executionID] = tr
	
	return tr
}

// StartSpan 开始跨度
func (t *tracerService) StartSpan(traceID, spanID, name string, parentSpanID *string) *trace.Span {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	span := trace.NewSpan(spanID, traceID, name)
	if parentSpanID != nil {
		span.SetParentSpanID(*parentSpanID)
	}
	
	t.spans[spanID] = span
	
	// 添加到追踪中
	if tr, exists := t.traces[traceID]; exists {
		tr.AddSpan(span)
	}
	
	return span
}

// EndSpan 结束跨度
func (t *tracerService) EndSpan(spanID string, status trace.SpanStatus) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	span, exists := t.spans[spanID]
	if !exists {
		return NewExecutionError("span not found: " + spanID)
	}
	
	span.End(status)
	
	// 添加到待刷新队列
	t.pendingSpans = append(t.pendingSpans, span)
	
	// 如果达到批量大小，立即刷新
	if len(t.pendingSpans) >= t.batchSize {
		return t.flushPendingSpans()
	}
	
	return nil
}

// AddSpanEvent 添加跨度事件
func (t *tracerService) AddSpanEvent(spanID, eventName string, attributes map[string]interface{}) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	span, exists := t.spans[spanID]
	if !exists {
		return NewExecutionError("span not found: " + spanID)
	}
	
	event := trace.NewEvent(eventName, attributes)
	span.AddEvent(event)
	
	return nil
}

// SetSpanAttribute 设置跨度属性
func (t *tracerService) SetSpanAttribute(spanID, key string, value interface{}) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	span, exists := t.spans[spanID]
	if !exists {
		return NewExecutionError("span not found: " + spanID)
	}
	
	span.SetAttribute(key, value)
	return nil
}

// FlushTraces 批量刷新追踪数据
func (t *tracerService) FlushTraces() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	return t.flushPendingSpans()
}

// flushPendingSpans 刷新待处理的跨度
func (t *tracerService) flushPendingSpans() error {
	if len(t.pendingSpans) == 0 {
		return nil
	}
	
	// 批量保存跨度
	for _, span := range t.pendingSpans {
		if err := t.repository.SaveSpan(span); err != nil {
			return err
		}
	}
	
	// 清空待刷新队列
	t.pendingSpans = t.pendingSpans[:0]
	
	return nil
}

// backgroundFlush 后台定期刷新
func (t *tracerService) backgroundFlush() {
	ticker := time.NewTicker(t.flushInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := t.FlushTraces(); err != nil {
				// 记录错误但不影响正常运行
			}
		case <-t.stopCh:
			// 最后一次刷新
			t.FlushTraces()
			return
		}
	}
}

// GetTrace 获取追踪
func (t *tracerService) GetTrace(traceID string) (*trace.Trace, error) {
	t.mutex.RLock()
	if tr, exists := t.traces[traceID]; exists {
		t.mutex.RUnlock()
		return tr, nil
	}
	t.mutex.RUnlock()
	
	// 从仓储加载
	spans, err := t.repository.FindTraceSpans(traceID)
	if err != nil {
		return nil, err
	}
	
	tr := trace.NewTrace(traceID)
	for _, span := range spans {
		tr.AddSpan(span)
	}
	
	// 缓存到内存
	t.mutex.Lock()
	t.traces[traceID] = tr
	t.mutex.Unlock()
	
	return tr, nil
}

// Close 关闭追踪器
func (t *tracerService) Close() error {
	close(t.stopCh)
	
	// 等待背景刷新完成
	time.Sleep(100 * time.Millisecond)
	
	return t.FlushTraces()
}
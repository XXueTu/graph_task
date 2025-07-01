package event

import (
	"fmt"
	"time"
	"github.com/XXueTu/graph_task/storage"
	"github.com/XXueTu/graph_task/types"
)

const (
	EventTracePublished = "trace.published"
	EventSpanStarted    = "span.started"
	EventSpanFinished   = "span.finished"
)

// TraceEventHandler 追踪事件处理器
type TraceEventHandler struct {
	storage storage.ExecutionStorage
}

// NewTraceEventHandler 创建追踪事件处理器
func NewTraceEventHandler(storage storage.ExecutionStorage) *TraceEventHandler {
	return &TraceEventHandler{storage: storage}
}

// HandleTracePublished 处理任务执行事件并创建span
func (h *TraceEventHandler) HandleTracePublished(event *Event) error {
	fmt.Printf("事件追踪处理: %v\n", event)
	
	// 检查input中是否包含trace_id
	var traceID string
	if input, exists := event.Data["input"]; exists {
		if inputMap, ok := input.(map[string]interface{}); ok {
			if tid, exists := inputMap["trace_id"]; exists {
				if tidStr, ok := tid.(string); ok {
					traceID = tidStr
				}
			}
		}
	}
	
	if traceID == "" {
		// 如果没有trace_id，不处理追踪
		return nil
	}
	
	// 获取任务ID
	taskID := event.TaskID
	if taskID == "" {
		return fmt.Errorf("missing task_id in event")
	}
	
	// 生成span ID
	spanID := fmt.Sprintf("span_%s_%s", event.ExecutionID, taskID)
	
	// 设置父span ID（根span）
	rootSpanID := fmt.Sprintf("span_root_%s", traceID)
	
	// 创建任务span
	span := &types.TraceSpan{
		SpanID:       spanID,
		TraceID:      traceID,
		ParentSpanID: &rootSpanID,
		Name:         taskID,
		StartTime:    event.Timestamp,
		Status:       "running",
		Attributes: map[string]interface{}{
			"task_id":      taskID,
			"execution_id": event.ExecutionID,
			"workflow_id":  event.WorkflowID,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// 从事件数据中获取更多信息
	if startTime, exists := event.Data["start_time"]; exists {
		if startTimeVal, ok := startTime.(int64); ok {
			span.StartTime = startTimeVal
		}
	}
	
	// 如果任务已完成，设置结束时间
	if endTime, exists := event.Data["end_time"]; exists {
		if endTimeVal, ok := endTime.(int64); ok {
			span.EndTime = &endTimeVal
			// 使用事件中的duration，或者计算差值
			if durationVal, exists := event.Data["duration"]; exists {
				if durationInt64, ok := durationVal.(int64); ok {
					// duration已经是毫秒，转换为纳秒
					durationNs := durationInt64 * 1000000
					span.Duration = &durationNs
				} else if durationFloat, ok := durationVal.(float64); ok {
					// duration是浮点数毫秒，转换为纳秒
					durationNs := int64(durationFloat * 1000000)
					span.Duration = &durationNs
				} else if durationInt, ok := durationVal.(int); ok {
					// duration是int毫秒，转换为纳秒
					durationNs := int64(durationInt) * 1000000
					span.Duration = &durationNs
				}
			} else {
				// 如果没有duration，计算差值（假设时间戳是毫秒）
				duration := (endTimeVal - span.StartTime) * 1000000 // 转换为纳秒
				span.Duration = &duration
			}
			span.Status = "completed"
			
			// 检查是否有错误
			if errorMsg, exists := event.Data["error"]; exists {
				if errorStr, ok := errorMsg.(string); ok && errorStr != "" {
					span.Status = "error"
					span.Attributes["error"] = errorStr
				}
			}
		}
	}
	
	// 添加输入输出到属性
	if input, exists := event.Data["input"]; exists {
		span.Attributes["input"] = input
	}
	if output, exists := event.Data["output"]; exists {
		span.Attributes["output"] = output
	}
	
	return h.storage.SaveSpan(span)
}

// HandleSpanStarted 处理Span开始事件
func (h *TraceEventHandler) HandleSpanStarted(event *Event) error {
	spanID, ok := event.Data["span_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid span_id in event data")
	}
	
	traceID, ok := event.Data["trace_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid trace_id in event data")
	}
	
	name, ok := event.Data["name"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid name in event data")
	}
	
	span := &types.TraceSpan{
		SpanID:    spanID,
		TraceID:   traceID,
		Name:      name,
		StartTime: event.Timestamp,
		Status:    "running",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// 设置父Span ID（如果存在）
	if parentSpanID, exists := event.Data["parent_span_id"]; exists {
		if parentSpanIDVal, ok := parentSpanID.(string); ok && parentSpanIDVal != "" {
			span.ParentSpanID = &parentSpanIDVal
		}
	}
	
	// 设置属性（如果存在）
	if attributes, exists := event.Data["attributes"]; exists {
		if attributesVal, ok := attributes.(map[string]interface{}); ok {
			span.Attributes = attributesVal
		}
	}
	
	return h.storage.SaveSpan(span)
}

// HandleSpanFinished 处理Span完成事件
func (h *TraceEventHandler) HandleSpanFinished(event *Event) error {
	spanID, ok := event.Data["span_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid span_id in event data")
	}
	
	// 获取现有的Span
	span, err := h.storage.GetSpan(spanID)
	if err != nil {
		return fmt.Errorf("failed to get span %s: %w", spanID, err)
	}
	
	// 更新结束时间和状态
	span.EndTime = &event.Timestamp
	duration := event.Timestamp - span.StartTime
	span.Duration = &duration
	span.UpdatedAt = time.Now()
	
	// 更新状态
	if status, exists := event.Data["status"]; exists {
		if statusVal, ok := status.(string); ok {
			span.Status = statusVal
		}
	} else {
		span.Status = "success"
	}
	
	// 更新属性（如果存在）
	if attributes, exists := event.Data["attributes"]; exists {
		if attributesVal, ok := attributes.(map[string]interface{}); ok {
			if span.Attributes == nil {
				span.Attributes = make(map[string]interface{})
			}
			for k, v := range attributesVal {
				span.Attributes[k] = v
			}
		}
	}
	
	// 添加事件（如果存在）
	if events, exists := event.Data["events"]; exists {
		if eventsVal, ok := events.([]types.TraceEvent); ok {
			span.Events = append(span.Events, eventsVal...)
		}
	}
	
	return h.storage.SaveSpan(span)
}

// 兼容性函数，保持原有的函数签名
func EventTracePublishedHandler(event *Event) error {
	fmt.Printf("事件追踪处理: %v\n", event)
	return nil
}

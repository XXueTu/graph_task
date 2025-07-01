package event

import (
	"fmt"
	"time"
	"github.com/XXueTu/graph_task/storage"
	"github.com/XXueTu/graph_task/types"
)

const (
	EventWorkflowPublished = "workflow.published"
	EventWorkflowCompleted = "workflow.completed"
)

func EventWorkflowPublishedHandler(event *Event) error {
	fmt.Printf("事件发布处理: %v\n", event)
	return nil
}

func EventWorkflowCompletedHandler(event *Event) error {
	fmt.Printf("事件执行完成处理: %v\n", event)
	return nil
}

// WorkflowEventHandler 工作流事件处理器
type WorkflowEventHandler struct {
	storage storage.ExecutionStorage
}

// NewWorkflowEventHandler 创建工作流事件处理器
func NewWorkflowEventHandler(storage storage.ExecutionStorage) *WorkflowEventHandler {
	return &WorkflowEventHandler{storage: storage}
}

// HandleWorkflowCompleted 处理工作流完成事件
func (h *WorkflowEventHandler) HandleWorkflowCompleted(event *Event) error {
	fmt.Printf("事件执行完成处理: %v\n", event)
	
	// 如果事件中包含trace相关信息，创建trace记录
	if traceID, exists := event.Data["trace_id"]; exists {
		if traceIDVal, ok := traceID.(string); ok {
			// 生成根span
			rootSpanID := fmt.Sprintf("span_%s_root", event.ExecutionID)
			
			// 创建主trace记录
			trace := &types.ExecutionTrace{
				TraceID:     traceIDVal,
				WorkflowID:  event.WorkflowID,
				ExecutionID: event.ExecutionID,
				RootSpanID:  rootSpanID,
				StartTime:   event.Timestamp,
				EndTime:     &event.Timestamp,
				Status:      "completed",
			}
			
			// 从事件数据中获取持续时间
			if duration, exists := event.Data["duration"]; exists {
				if durationVal, ok := duration.(int64); ok {
					trace.Duration = &durationVal
					startTime := event.Timestamp - durationVal
					trace.StartTime = startTime
				}
			}
			
			// 保存trace
			if err := h.storage.SaveTrace(trace); err != nil {
				fmt.Printf("Failed to save trace: %v\n", err)
			}
			
			// 创建根span
			rootSpan := &types.TraceSpan{
				SpanID:    rootSpanID,
				TraceID:   traceIDVal,
				Name:      "workflow-execution",
				StartTime: trace.StartTime,
				EndTime:   &event.Timestamp,
				Duration:  trace.Duration,
				Status:    "completed",
				Attributes: map[string]interface{}{
					"workflow_id":  event.WorkflowID,
					"execution_id": event.ExecutionID,
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			
			// 添加事件数据作为属性
			for k, v := range event.Data {
				if k != "trace_id" && k != "duration" {
					rootSpan.Attributes[k] = v
				}
			}
			
			// 保存根span
			if err := h.storage.SaveSpan(rootSpan); err != nil {
				fmt.Printf("Failed to save root span: %v\n", err)
			}
		}
	}
	
	return nil
}

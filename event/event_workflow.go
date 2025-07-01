package event

import "fmt"

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

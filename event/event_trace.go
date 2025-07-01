package event

import "fmt"

const (
	EventTracePublished = "trace.published"
)

func EventTracePublishedHandler(event *Event) error {
	fmt.Printf("事件追踪处理: %v\n", event)
	return nil
}

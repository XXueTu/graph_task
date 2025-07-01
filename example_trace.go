package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/XXueTu/graph_task/engine"
	"github.com/XXueTu/graph_task/event"
	"github.com/XXueTu/graph_task/storage"
	"github.com/XXueTu/graph_task/types"
)

func main() {
	// Create storage for trace persistence
	store, err := storage.NewMySQLExecutionStorage("root:Root@123@tcp(10.99.51.9:3306)/graph_task?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		log.Printf("Warning: Could not connect to MySQL, using memory storage: %v", err)
		// In a real scenario, you would implement a memory storage
		return
	}
	defer store.Close()

	// Create engine with trace-enabled event handlers
	engine := engine.NewEngine(
		engine.WithStorage(store),
		engine.WithMaxConcurrency(10),
	)

	// Create trace event handlers
	traceHandler := event.NewTraceEventHandler(store)
	workflowHandler := event.NewWorkflowEventHandler(store)

	// Subscribe to trace events
	engine.Subscribe(event.EventTracePublished, traceHandler.HandleTracePublished)
	engine.Subscribe(event.EventSpanStarted, traceHandler.HandleSpanStarted)
	engine.Subscribe(event.EventSpanFinished, traceHandler.HandleSpanFinished)
	engine.Subscribe(event.EventWorkflowCompleted, workflowHandler.HandleWorkflowCompleted)

	// Create a simple workflow
	builder := engine.CreateWorkflow("trace-example")
	builder.SetDescription("Example workflow with trace persistence")
	builder.AddTask("step1", "Data Processing", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		// Simulate processing
		time.Sleep(100 * time.Millisecond)
		return map[string]interface{}{
			"processed_data": fmt.Sprintf("processed_%s", input["raw_data"]),
		}, nil
	})
	builder.AddTask("step2", "Data Transform", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		// Simulate transformation
		time.Sleep(150 * time.Millisecond)
		return map[string]interface{}{
			"transformed_data": fmt.Sprintf("transformed_%s", input["processed_data"]),
		}, nil
	})
	builder.AddTask("step3", "Data Cleanup", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		// Simulate cleanup
		time.Sleep(80 * time.Millisecond)
		return map[string]interface{}{
			"final_output": fmt.Sprintf("output_%s", input["transformed_data"]),
		}, nil
	})

	// Add dependencies
	builder.AddDependency("step1", "step2")
	builder.AddDependency("step2", "step3")

	workflow, err := builder.Build()
	if err != nil {
		log.Fatal(err)
	}

	err = engine.PublishWorkflow(workflow)
	if err != nil {
		log.Fatal(err)
	}

	// Execute workflow with trace data
	ctx := context.Background()
	traceID := fmt.Sprintf("trace_%d", time.Now().UnixNano())

	// Simulate creating a trace before execution
	trace := &types.ExecutionTrace{
		TraceID:     traceID,
		WorkflowID:  workflow.ID,
		ExecutionID: "exec_" + traceID,
		RootSpanID:  "span_root_" + traceID,
		StartTime:   time.Now().UnixNano(),
		Status:      "running",
	}

	// Save initial trace
	if err := store.SaveTrace(trace); err != nil {
		log.Printf("Error saving trace: %v", err)
	}

	// Create root span
	rootSpan := &types.TraceSpan{
		SpanID:    trace.RootSpanID,
		TraceID:   traceID,
		Name:      "workflow-execution",
		StartTime: trace.StartTime,
		Status:    "running",
		Attributes: map[string]interface{}{
			"workflow_id":   workflow.ID,
			"workflow_name": workflow.Name,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.SaveSpan(rootSpan); err != nil {
		log.Printf("Error saving root span: %v", err)
	}

	// Execute workflow
	input := map[string]interface{}{
		"raw_data": "user_input",
		"trace_id": traceID,
	}

	result, err := engine.Execute(ctx, workflow.ID, input)
	if err != nil {
		log.Printf("Execution failed: %v", err)
		return
	}

	// Update trace with completion
	endTime := time.Now().UnixNano()
	trace.EndTime = &endTime
	duration := endTime - trace.StartTime
	trace.Duration = &duration
	trace.Status = "completed"

	if err := store.SaveTrace(trace); err != nil {
		log.Printf("Error updating trace: %v", err)
	}

	// Update root span
	rootSpan.EndTime = &endTime
	rootSpan.Duration = &duration
	rootSpan.Status = "completed"
	rootSpan.UpdatedAt = time.Now()

	if err := store.SaveSpan(rootSpan); err != nil {
		log.Printf("Error updating root span: %v", err)
	}

	log.Printf("Workflow execution completed: %s", result.ExecutionID)
	log.Printf("Execution status: %s", result.Status)

	// Demonstrate trace retrieval
	log.Println("\n=== Trace Data Retrieved from Database ===")

	// Get the trace
	retrievedTrace, err := store.GetTrace(traceID)
	if err != nil {
		log.Printf("Error retrieving trace: %v", err)
	} else {
		log.Printf("Trace ID: %s", retrievedTrace.TraceID)
		log.Printf("Workflow ID: %s", retrievedTrace.WorkflowID)
		log.Printf("Duration: %dms", *retrievedTrace.Duration/1000000)
		log.Printf("Status: %s", retrievedTrace.Status)
	}

	// Get trace spans
	spans, err := store.GetTraceSpans(traceID)
	if err != nil {
		log.Printf("Error retrieving spans: %v", err)
	} else {
		log.Printf("\nSpans (%d):", len(spans))
		for _, span := range spans {
			log.Printf("  - %s (%s) [%dms]", span.Name, span.Status,
				func() int64 {
					if span.Duration != nil {
						return *span.Duration / 1000000
					}
					return 0
				}())
		}
	}

	// List recent traces for this workflow
	traces, err := store.ListTraces(workflow.ID, 0, 10)
	if err != nil {
		log.Printf("Error listing traces: %v", err)
	} else {
		log.Printf("\nRecent traces for workflow %s:", workflow.ID)
		for _, t := range traces {
			log.Printf("  - %s (%s)", t.TraceID, t.Status)
		}
	}
}

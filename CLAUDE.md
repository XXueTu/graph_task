# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a high-performance task orchestration and execution system built in Go (`github.com/XXueTu/graph_task`). It provides:

- **Memory-Only Workflow Registry**: Workflows registered at startup and kept in memory only
- **Storage Interface**: Flexible storage interface supporting MySQL for execution records, logs, and tracking data
- **High-Performance Execution Engine**: DAG-based concurrent task execution with automatic retry
- **Execution Tracking**: High-performance execution tracing with batch processing and monitoring
- **Manual Retry Management**: Failed executions can be manually retried through management interface
- **Event-Driven Architecture**: Event bus for workflow and execution lifecycle events
- **SDK Integration**: Go SDK for client integration

## Development Commands

### Go Module Management
```bash
# Initialize dependencies
go mod tidy

# Download dependencies
go mod download

# Verify dependencies
go mod verify
```

### Building and Running
```bash
# Build the project
go build ./...

# Run tests
go test ./...

# Run specific test
go test -run TestBasicWorkflow

# Run benchmark tests
go test -bench=.

# Build for production
go build -o task_graph .
```

### Code Quality
```bash
# Format code
go fmt ./...

# Run static analysis
go vet ./...

# Run linter (if golangci-lint is installed)
golangci-lint run
```

## Architecture Overview

### Core Components

1. **Engine** (`engine.go`): Core orchestration engine
   - In-memory workflow registry (workflows stored in memory only)
   - Automatic retry with exponential backoff
   - Manual retry management for failed executions
   - Execution tracking and monitoring integration
   - Event-driven architecture with event bus

2. **ExecutionStorage** (`execution_storage.go`): Storage abstraction layer
   - Interface for execution records persistence
   - Task execution result tracking
   - Support for MySQL and other storage backends
   - Execution logs and traces storage

3. **ExecutionTracer** (`execution_tracer.go`): High-performance execution tracking
   - Batch processing for optimal performance
   - Span-based operation tracking with async storage
   - Execution logs and metrics collection
   - Memory management with automatic cleanup
   - Langfuse-inspired tracing design

4. **RetryManager** (`retry_manager.go`): Comprehensive retry management
   - Automatic retry with configurable exponential backoff
   - Manual retry capability for failed executions
   - Retry statistics and tracking
   - Failed execution recovery across instances

5. **Builder** (`builder.go`): DSL workflow builder
   - Fluent API for workflow definition
   - Dependency validation and circular dependency detection
   - Task configuration and relationship management

6. **Planner** (`planner.go`): Execution plan optimization
   - DAG analysis and stage generation
   - Concurrency optimization
   - Execution plan caching support

7. **Executor** (`executor.go`): High-performance task executor
   - Worker pool management
   - Concurrent task execution with context support
   - Integration with execution tracing

8. **ContextManager** (`context_manager.go`): Local execution context management
   - In-memory execution state management
   - Automatic cleanup of old contexts
   - Task result coordination and data sharing

9. **EventBus** (`eventbus.go`): Event-driven messaging system
   - Publish-subscribe pattern for workflow events
   - Lifecycle event handling
   - Extensible event handling system

10. **SDK** (`sdk.go`): Client integration SDK
    - Workflow registration and management
    - Execution management and monitoring
    - Failed execution retry interface

### Key Data Structures

- **Workflow**: In-memory workflow definition with tasks and dependencies (never persisted to storage)
- **Task**: Individual task with handler, timeout, retry configuration, and execution state
- **ExecutionPlan**: Optimized execution plan with staged execution and concurrency control
- **ExecutionResult**: Comprehensive execution results with task-level tracking and retry information
- **ExecutionTrace**: High-performance execution trace with spans, logs, and batch processing
- **RetryInfo**: Failed execution information for manual retry management across instances
- **ExecutionContext**: Local in-memory runtime context for execution coordination
- **Event**: Event-driven system events for workflow and execution lifecycle

### Performance Optimizations

1. **Memory-Only Workflows**: Zero storage overhead for workflow definitions
2. **Batch Processing**: Async batch processing for execution traces and logs to minimize I/O
3. **Local Context Management**: Fast in-memory execution state without persistence overhead
4. **Automatic Retry**: Built-in exponential backoff retry without user intervention
5. **Execution-Only Storage**: Storage interface only handles execution data, not workflow definitions
6. **High-Performance Tracing**: Efficient span-based execution monitoring with batch storage
7. **Thread-Safe Operations**: Concurrent-safe context and state management
8. **Event-Driven Architecture**: Async event processing for system decoupling

## Usage Examples

### Basic Engine Setup
```go
// Create engine with storage and retry configuration
storage, _ := NewMySQLExecutionStorage("user:pass@tcp(localhost:3306)/taskdb")
engine := NewEngine(
    WithStorage(storage),
    WithRetryConfig(&RetryConfig{
        MaxAutoRetries:    3,
        RetryDelay:        1 * time.Second,
        BackoffMultiplier: 2.0,
        MaxRetryDelay:     30 * time.Second,
    }),
    WithMaxConcurrency(10),
)

// Build and publish workflow (stored in memory only)
builder := engine.CreateWorkflow("example-workflow")
builder.SetDescription("Example Processing Pipeline")
builder.AddTask("task1", "Process Data", processHandler)
builder.AddTask("task2", "Cleanup", cleanupHandler)
builder.AddDependency("task1", "task2")

workflow, err := builder.Build()
if err != nil {
    log.Fatal(err)
}

err = engine.PublishWorkflow(workflow)
if err != nil {
    log.Fatal(err)
}
```

### Workflow Execution with Automatic Retry
```go
// Execute workflow (automatic retry built-in)
input := map[string]interface{}{
    "data": "test input",
    "config": map[string]interface{}{
        "setting1": "value1",
    },
}

result, err := engine.Execute(ctx, "example-workflow", input)
if err != nil {
    // Execution failed after all automatic retries
    log.Printf("Execution failed: %v", err)
} else {
    log.Printf("Execution completed: %s, Status: %s", result.ExecutionID, result.Status)
}

// Check execution status and logs
status, err := engine.GetExecutionStatus(result.ExecutionID)
if err == nil {
    log.Printf("Progress: %.1f%%, Status: %s", status.Progress, status.Status)
}

// Get detailed execution logs
logs, err := engine.GetExecutionLogs(result.ExecutionID)
if err == nil {
    for _, log := range logs {
        fmt.Printf("[%s] %s: %s\n", log.Level, log.Timestamp, log.Message)
    }
}
```

### Manual Retry Management
```go
// List failed executions that can be manually retried
failedExecutions, err := engine.GetFailedExecutions()
for _, failed := range failedExecutions {
    log.Printf("Failed: %s, Reason: %s, Attempts: %d", 
        failed.ExecutionID, failed.FailureReason, len(failed.ManualRetries))
}

// Manually retry a failed execution
result, err := engine.ManualRetry(ctx, "failed-execution-id")
if err != nil {
    log.Printf("Manual retry failed: %v", err)
} else {
    log.Printf("Manual retry successful: %s", result.ExecutionID)
}
```

### Async Execution and Monitoring
```go
// Execute workflow asynchronously
executionID, err := engine.ExecuteAsync(ctx, "example-workflow", input)
if err != nil {
    log.Printf("Failed to start async execution: %v", err)
    return
}

log.Printf("Started async execution: %s", executionID)

// Monitor execution progress
ticker := time.NewTicker(5 * time.Second)
defer ticker.Stop()

for {
    select {
    case <-ticker.C:
        result, err := engine.GetExecutionResult(executionID)
        if err != nil {
            log.Printf("Error getting execution result: %v", err)
            continue
        }
        
        log.Printf("Execution %s status: %s", executionID, result.Status)
        
        if result.Status == WorkflowStatusSuccess || result.Status == WorkflowStatusFailed {
            log.Printf("Execution completed with status: %s", result.Status)
            return
        }
    case <-ctx.Done():
        log.Printf("Context cancelled, stopping monitoring")
        return
    }
}
```

### Engine with Automatic and Manual Retry
```go
// Create engine with retry configuration
engine := NewEngine(
    WithRetryConfig(&RetryConfig{
        MaxAutoRetries:    3,                // Maximum 3 automatic retries
        RetryDelay:        2 * time.Second,  // Initial delay 2 seconds
        BackoffMultiplier: 2.0,              // Exponential backoff multiplier
        MaxRetryDelay:     30 * time.Second, // Maximum delay 30 seconds
    }),
    WithMaxConcurrency(10),
)

// Create workflow with automatic retry support
builder := engine.CreateWorkflow("retry-example")
builder.AddTask("flaky-task", "Process Data", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    // Task implementation - failures will be automatically retried with exponential backoff
    return map[string]interface{}{"result": "processed"}, nil
})

workflow, err := builder.Build()
if err != nil {
    log.Fatal(err)
}

// Publish workflow (stored in memory only)
err = engine.PublishWorkflow(workflow)

// Execute with automatic retry
result, err := engine.Execute(ctx, workflow.ID, map[string]interface{}{
    "input": "test data",
})

if err != nil {
    // If automatic retries failed, check for manual retry options
    failedExecutions, _ := engine.GetFailedExecutions()
    for _, failed := range failedExecutions {
        log.Printf("Failed execution: %s, reason: %s", failed.ExecutionID, failed.FailureReason)
        
        // Manual retry failed execution
        retryResult, retryErr := engine.ManualRetry(ctx, failed.ExecutionID)
        if retryErr == nil {
            log.Printf("Manual retry succeeded: %s", retryResult.ExecutionID)
        }
    }
}

// Get retry statistics
stats := engine.GetRetryStatistics()
log.Printf("Retry stats - Total Failed: %d, Recovered: %d, Pending: %d", 
    stats.TotalFailed, stats.Recovered, stats.PendingRetry)
```

## Configuration Options

### Engine Configuration
- `WithStorage(storage)`: Set storage backend for execution data (supports MySQL and other backends)
- `WithRetryConfig(config)`: Configure automatic retry behavior and exponential backoff
- `WithRetryManager(manager)`: Set custom retry manager for failed execution handling
- `WithMaxConcurrency(n)`: Set maximum concurrency for task execution
- `WithContextManager(manager)`: Set local context manager for execution coordination  
- `WithExecutionTracer(tracer)`: Set custom execution tracer for monitoring and logging

### RetryConfig Options
- `MaxAutoRetries`: Maximum automatic retry attempts (default: 3)
- `RetryDelay`: Initial retry delay (default: 1s)
- `BackoffMultiplier`: Exponential backoff multiplier (default: 2.0)
- `MaxRetryDelay`: Maximum retry delay cap (default: 30s)

### Retry Features
- **Automatic Retry**: Built-in exponential backoff retry for failed executions
- **Manual Retry**: Interface for manually retrying failed executions
- **Retry Statistics**: Track retry success rates and failure patterns
- **Persistent Retry Info**: Failed executions stored for cross-instance manual retry
- **Retry Abandonment**: Mark failed executions as abandoned to stop retry attempts

### Storage Interface Implementation
- Stores execution records, task results, and execution metadata
- Workflow definitions are never persisted to storage (memory-only)
- Extensible interface supporting MySQL and other storage backends
- Automatic retry information persistence for cross-instance manual retry

### Task Configuration
- Timeout: Task execution timeout duration
- Retry: Handled automatically by engine retry logic with exponential backoff
- Dependencies: Task dependency relationships for execution ordering
- Handler: Task execution function (not serializable, stored in memory only)
- Input/Output: Task-specific data transformation configuration

## Testing

The project includes comprehensive tests:
- Unit tests for all components
- Integration tests for end-to-end workflows
- Performance benchmark tests
- Error handling tests
- Concurrency tests

Run specific test suites:
```bash
go test -run TestBasicWorkflow
go test -run TestRetry
go test -run TestParallel 
go test -bench=BenchmarkWorkflowExecution
go test -v ./... # Run all tests with verbose output
```

## Deployment Considerations

### Multi-Instance Deployment
- **Workflow Registration**: Each instance must register workflows at startup (memory-only)
- **Shared Storage**: All instances share storage backend for execution records
- **Local Context**: Each instance manages its own execution contexts separately
- **Event Bus**: Event-driven architecture supports distributed event handling
- **Manual Retry Interface**: Failed executions can be retried from any instance

### Performance Tuning
- **Memory Management**: Monitor workflow registry memory usage per instance
- **Retry Configuration**: Tune automatic retry settings based on failure patterns
- **Execution Tracking**: Configure batch processing parameters for optimal performance
- **Concurrency Settings**: Adjust max concurrency based on system resources
- **Context Cleanup**: Tune local context cleanup intervals for optimal memory usage

### Storage Requirements
- **Primary Storage**: Execution records, task results, and execution metadata
  - Execution logs and traces with batch processing
  - Retry information and manual retry records
  - No workflow definitions stored (memory-only)
- **Memory**: Workflow definitions and active execution contexts
- **Cleanup**: Regular cleanup of old execution records and contexts

### Startup Requirements
- Workflows must be registered during application startup
- Storage connectivity required for execution data persistence
- Event bus initialization for lifecycle event handling
- Failed execution recovery interface should be available for manual retries

## Go Version

The project uses Go 1.23.0 as specified in go.mod and leverages modern Go features including generics and improved performance optimizations.
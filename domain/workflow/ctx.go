package workflow

import (
	"context"

	"github.com/XXueTu/graph_task/domain/logger"
)

// 自定义context
type ExecContext struct {
	Context context.Context
	Logger  logger.LoggerService
}

func NewExecContext(ctx context.Context, executionID, taskID string, log logger.LoggerService) ExecContext {
	ctxWithExecID := context.WithValue(ctx, logger.ExecutionIDKey, executionID)
	ctxWithTaskID := context.WithValue(ctxWithExecID, logger.TaskIDKey, taskID)
	return ExecContext{Context: ctxWithTaskID, Logger: log}
}

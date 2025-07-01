package workflow

import "fmt"

// WorkflowError 工作流领域错误
type WorkflowError struct {
	message string
}

func (e *WorkflowError) Error() string {
	return e.message
}

// NewWorkflowError 创建工作流错误
func NewWorkflowError(message string) *WorkflowError {
	return &WorkflowError{message: message}
}

// NewWorkflowErrorf 创建格式化工作流错误
func NewWorkflowErrorf(format string, args ...interface{}) *WorkflowError {
	return &WorkflowError{message: fmt.Sprintf(format, args...)}
}

// IsWorkflowError 判断是否为工作流错误
func IsWorkflowError(err error) bool {
	_, ok := err.(*WorkflowError)
	return ok
}
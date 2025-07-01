package execution

import "fmt"

// ExecutionError 执行领域错误
type ExecutionError struct {
	message string
}

func (e *ExecutionError) Error() string {
	return e.message
}

// NewExecutionError 创建执行错误
func NewExecutionError(message string) *ExecutionError {
	return &ExecutionError{message: message}
}

// NewExecutionErrorf 创建格式化执行错误
func NewExecutionErrorf(format string, args ...interface{}) *ExecutionError {
	return &ExecutionError{message: fmt.Sprintf(format, args...)}
}

// IsExecutionError 判断是否为执行错误
func IsExecutionError(err error) bool {
	_, ok := err.(*ExecutionError)
	return ok
}
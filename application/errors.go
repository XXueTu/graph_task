package application

import "fmt"

// ApplicationError 应用层错误
type ApplicationError struct {
	message string
}

func (e *ApplicationError) Error() string {
	return e.message
}

// NewApplicationError 创建应用错误
func NewApplicationError(message string) *ApplicationError {
	return &ApplicationError{message: message}
}

// NewApplicationErrorf 创建格式化应用错误
func NewApplicationErrorf(format string, args ...interface{}) *ApplicationError {
	return &ApplicationError{message: fmt.Sprintf(format, args...)}
}

// IsApplicationError 判断是否为应用错误
func IsApplicationError(err error) bool {
	_, ok := err.(*ApplicationError)
	return ok
}
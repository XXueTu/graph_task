package memory

import "fmt"

// RepositoryError 仓储错误
type RepositoryError struct {
	message string
}

func (e *RepositoryError) Error() string {
	return e.message
}

// NewRepositoryError 创建仓储错误
func NewRepositoryError(message string) *RepositoryError {
	return &RepositoryError{message: message}
}

// NewRepositoryErrorf 创建格式化仓储错误
func NewRepositoryErrorf(format string, args ...interface{}) *RepositoryError {
	return &RepositoryError{message: fmt.Sprintf(format, args...)}
}

// IsRepositoryError 判断是否为仓储错误
func IsRepositoryError(err error) bool {
	_, ok := err.(*RepositoryError)
	return ok
}
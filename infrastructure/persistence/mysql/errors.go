package mysql

import "fmt"

// MySQLError MySQL错误
type MySQLError struct {
	message string
}

func (e *MySQLError) Error() string {
	return e.message
}

// NewMySQLError 创建MySQL错误
func NewMySQLError(message string) *MySQLError {
	return &MySQLError{message: message}
}

// NewMySQLErrorf 创建格式化MySQL错误
func NewMySQLErrorf(format string, args ...interface{}) *MySQLError {
	return &MySQLError{message: fmt.Sprintf(format, args...)}
}

// IsMySQLError 判断是否为MySQL错误
func IsMySQLError(err error) bool {
	_, ok := err.(*MySQLError)
	return ok
}
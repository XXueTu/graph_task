package execution

// Repository 执行仓储接口
type Repository interface {
	// SaveExecution 保存执行记录
	SaveExecution(execution *Execution) error
	
	// FindExecutionByID 根据ID查找执行
	FindExecutionByID(id string) (*Execution, error)
	
	// ListExecutions 列出执行记录
	ListExecutions(workflowID string, limit, offset int) ([]*Execution, error)
	
	// UpdateExecution 更新执行记录
	UpdateExecution(execution *Execution) error
	
	// DeleteExecution 删除执行记录
	DeleteExecution(id string) error
	
	// ListRunningExecutions 列出正在运行的执行
	ListRunningExecutions() ([]*Execution, error)
}
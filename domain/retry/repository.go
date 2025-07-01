package retry

// Repository 重试仓储接口
type Repository interface {
	// SaveRetryInfo 保存重试信息
	SaveRetryInfo(info *Info) error
	
	// FindRetryInfo 根据执行ID查找重试信息
	FindRetryInfo(executionID string) (*Info, error)
	
	// ListFailedExecutions 列出失败的执行
	ListFailedExecutions() ([]*Info, error)
	
	// UpdateRetryInfo 更新重试信息
	UpdateRetryInfo(info *Info) error
	
	// DeleteRetryInfo 删除重试信息
	DeleteRetryInfo(executionID string) error
	
	// GetStatistics 获取重试统计
	GetStatistics() (*Statistics, error)
}
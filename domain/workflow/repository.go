package workflow

// Repository 工作流仓储接口（内存实现）
type Repository interface {
	// Save 保存工作流（仅内存）
	Save(workflow *Workflow) error
	
	// FindByID 根据ID查找工作流
	FindByID(id string) (*Workflow, error)
	
	// FindAll 查找所有工作流
	FindAll() ([]*Workflow, error)
	
	// Delete 删除工作流
	Delete(id string) error
	
	// Exists 检查工作流是否存在
	Exists(id string) bool
}
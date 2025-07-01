package memory

import (
	"sync"

	"github.com/XXueTu/graph_task/domain/workflow"
)

// workflowRepository 内存工作流仓储实现
type workflowRepository struct {
	workflows map[string]*workflow.Workflow
	mutex     sync.RWMutex
}

// NewWorkflowRepository 创建内存工作流仓储
func NewWorkflowRepository() workflow.Repository {
	return &workflowRepository{
		workflows: make(map[string]*workflow.Workflow),
	}
}

// Save 保存工作流
func (r *workflowRepository) Save(wf *workflow.Workflow) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.workflows[wf.ID()] = wf
	return nil
}

// FindByID 根据ID查找工作流
func (r *workflowRepository) FindByID(id string) (*workflow.Workflow, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	wf, exists := r.workflows[id]
	if !exists {
		return nil, NewRepositoryError("workflow not found: " + id)
	}
	
	return wf, nil
}

// FindAll 查找所有工作流
func (r *workflowRepository) FindAll() ([]*workflow.Workflow, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	workflows := make([]*workflow.Workflow, 0, len(r.workflows))
	for _, wf := range r.workflows {
		workflows = append(workflows, wf)
	}
	
	return workflows, nil
}

// Delete 删除工作流
func (r *workflowRepository) Delete(id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if _, exists := r.workflows[id]; !exists {
		return NewRepositoryError("workflow not found: " + id)
	}
	
	delete(r.workflows, id)
	return nil
}

// Exists 检查工作流是否存在
func (r *workflowRepository) Exists(id string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	_, exists := r.workflows[id]
	return exists
}
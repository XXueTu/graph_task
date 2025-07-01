package application

import (
	"github.com/XXueTu/graph_task/domain/workflow"
)

// WorkflowService 工作流应用服务
type WorkflowService struct {
	workflowRepo workflow.Repository
	eventPub     EventPublisher
}

// NewWorkflowService 创建工作流服务
func NewWorkflowService(workflowRepo workflow.Repository, eventPub EventPublisher) *WorkflowService {
	return &WorkflowService{
		workflowRepo: workflowRepo,
		eventPub:     eventPub,
	}
}

// CreateWorkflow 创建工作流
func (s *WorkflowService) CreateWorkflow(id, name string) workflow.Builder {
	return workflow.NewBuilder(id).SetName(name)
}

// PublishWorkflow 发布工作流
func (s *WorkflowService) PublishWorkflow(wf *workflow.Workflow) error {
	// 发布工作流
	if err := wf.Publish(); err != nil {
		return err
	}
	
	// 保存到仓储
	if err := s.workflowRepo.Save(wf); err != nil {
		return err
	}
	
	// 发布事件
	event := NewWorkflowEvent("workflow.published", wf.ID(), map[string]interface{}{
		"name":        wf.Name(),
		"description": wf.Description(),
		"version":     wf.Version(),
	})
	s.eventPub.Publish(event)
	
	return nil
}

// GetWorkflow 获取工作流
func (s *WorkflowService) GetWorkflow(id string) (*workflow.Workflow, error) {
	return s.workflowRepo.FindByID(id)
}

// ListWorkflows 列出所有工作流
func (s *WorkflowService) ListWorkflows() ([]*workflow.Workflow, error) {
	return s.workflowRepo.FindAll()
}

// DeleteWorkflow 删除工作流
func (s *WorkflowService) DeleteWorkflow(id string) error {
	// 检查工作流是否存在
	if !s.workflowRepo.Exists(id) {
		return NewApplicationError("workflow not found: " + id)
	}
	
	// 删除工作流
	if err := s.workflowRepo.Delete(id); err != nil {
		return err
	}
	
	// 发布事件
	event := NewWorkflowEvent("workflow.deleted", id, map[string]interface{}{})
	s.eventPub.Publish(event)
	
	return nil
}

// ValidateWorkflow 验证工作流
func (s *WorkflowService) ValidateWorkflow(wf *workflow.Workflow) error {
	// 工作流自身验证在Publish方法中进行
	return nil
}
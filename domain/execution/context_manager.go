package execution

import (
	"context"
	"sync"
	"time"
)

// ContextManagerService 执行上下文管理服务
type ContextManagerService interface {
	// CreateContext 创建执行上下文
	CreateContext(executionID, workflowID string) *Context

	// GetContext 获取执行上下文
	GetContext(executionID string) (*Context, error)

	// RemoveContext 移除执行上下文
	RemoveContext(executionID string)

	// CleanupOldContexts 清理旧的上下文
	CleanupOldContexts(maxAge time.Duration)

	// GetActiveContextCount 获取活跃上下文数量
	GetActiveContextCount() int

	// ListActiveContexts 列出活跃上下文
	ListActiveContexts() []string

	// Close 关闭上下文管理器
	Close() error
}

// contextManagerService 上下文管理服务实现
type contextManagerService struct {
	contexts        map[string]*Context
	contextAges     map[string]time.Time
	cleanupInterval time.Duration
	maxContextAge   time.Duration
	mutex           sync.RWMutex
	stopCh          chan struct{}
}

// NewContextManagerService 创建上下文管理服务
func NewContextManagerService(cleanupInterval, maxContextAge time.Duration) ContextManagerService {
	service := &contextManagerService{
		contexts:        make(map[string]*Context),
		contextAges:     make(map[string]time.Time),
		cleanupInterval: cleanupInterval,
		maxContextAge:   maxContextAge,
		stopCh:          make(chan struct{}),
	}

	// 启动清理协程
	go service.backgroundCleanup()

	return service
}

// CreateContext 创建执行上下文
func (c *contextManagerService) CreateContext(executionID, workflowID string) *Context {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	ctx := NewContext(context.TODO(), executionID, workflowID, nil)
	c.contexts[executionID] = ctx
	c.contextAges[executionID] = time.Now()

	return ctx
}

// GetContext 获取执行上下文
func (c *contextManagerService) GetContext(executionID string) (*Context, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	ctx, exists := c.contexts[executionID]
	if !exists {
		return nil, NewExecutionError("context not found: " + executionID)
	}

	return ctx, nil
}

// RemoveContext 移除执行上下文
func (c *contextManagerService) RemoveContext(executionID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if ctx, exists := c.contexts[executionID]; exists {
		ctx.Cancel()
		delete(c.contexts, executionID)
		delete(c.contextAges, executionID)
	}
}

// CleanupOldContexts 清理旧的上下文
func (c *contextManagerService) CleanupOldContexts(maxAge time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for executionID, createdAt := range c.contextAges {
		if now.Sub(createdAt) > maxAge {
			if ctx, exists := c.contexts[executionID]; exists {
				ctx.Cancel()
			}
			delete(c.contexts, executionID)
			delete(c.contextAges, executionID)
		}
	}
}

// GetActiveContextCount 获取活跃上下文数量
func (c *contextManagerService) GetActiveContextCount() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.contexts)
}

// ListActiveContexts 列出活跃上下文
func (c *contextManagerService) ListActiveContexts() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	contexts := make([]string, 0, len(c.contexts))
	for executionID := range c.contexts {
		contexts = append(contexts, executionID)
	}

	return contexts
}

// backgroundCleanup 后台清理协程
func (c *contextManagerService) backgroundCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.CleanupOldContexts(c.maxContextAge)
		case <-c.stopCh:
			return
		}
	}
}

// Close 关闭上下文管理器
func (c *contextManagerService) Close() error {
	close(c.stopCh)

	// 取消所有上下文
	c.mutex.Lock()
	for _, ctx := range c.contexts {
		ctx.Cancel()
	}
	c.contexts = make(map[string]*Context)
	c.contextAges = make(map[string]time.Time)
	c.mutex.Unlock()

	return nil
}

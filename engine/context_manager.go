package engine

import (
	"sync"
	"time"
)

// LocalContextManager 本地执行上下文管理器
type LocalContextManager struct {
	contexts map[string]*ExecutionContext
	mutex    sync.RWMutex

	// 清理相关
	cleanupInterval time.Duration
	maxAge          time.Duration
	ticker          *time.Ticker
	done            chan bool
}

// NewLocalContextManager 创建本地上下文管理器
func NewLocalContextManager(cleanupInterval, maxAge time.Duration) *LocalContextManager {
	if cleanupInterval <= 0 {
		cleanupInterval = 5 * time.Minute
	}
	if maxAge <= 0 {
		maxAge = 30 * time.Minute
	}

	lcm := &LocalContextManager{
		contexts:        make(map[string]*ExecutionContext),
		cleanupInterval: cleanupInterval,
		maxAge:          maxAge,
		ticker:          time.NewTicker(cleanupInterval),
		done:            make(chan bool),
	}

	// 启动清理协程
	go lcm.cleanup()

	return lcm
}

// CreateContext 创建执行上下文
func (lcm *LocalContextManager) CreateContext(executionID, workflowID string, input map[string]any) *ExecutionContext {
	lcm.mutex.Lock()
	defer lcm.mutex.Unlock()

	ctx := &ExecutionContext{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		GlobalData:  input,
		TaskResults: make(map[string]*TaskExecutionResult),
		createdAt:   time.Now(), // 添加创建时间用于清理
	}

	lcm.contexts[executionID] = ctx
	return ctx
}

// GetContext 获取执行上下文
func (lcm *LocalContextManager) GetContext(executionID string) (*ExecutionContext, bool) {
	lcm.mutex.RLock()
	defer lcm.mutex.RUnlock()

	ctx, exists := lcm.contexts[executionID]
	return ctx, exists
}

// UpdateContext 更新执行上下文
func (lcm *LocalContextManager) UpdateContext(executionID string, ctx *ExecutionContext) {
	lcm.mutex.Lock()
	defer lcm.mutex.Unlock()

	lcm.contexts[executionID] = ctx
}

// RemoveContext 移除执行上下文
func (lcm *LocalContextManager) RemoveContext(executionID string) {
	lcm.mutex.Lock()
	defer lcm.mutex.Unlock()

	if ctx, exists := lcm.contexts[executionID]; exists {
		// 取消上下文
		if ctx.cancel != nil {
			ctx.cancel()
		}
		delete(lcm.contexts, executionID)
	}
}

// GetActiveContexts 获取活跃上下文数量
func (lcm *LocalContextManager) GetActiveContexts() int {
	lcm.mutex.RLock()
	defer lcm.mutex.RUnlock()

	return len(lcm.contexts)
}

// GetContextsByWorkflow 获取指定工作流的所有上下文
func (lcm *LocalContextManager) GetContextsByWorkflow(workflowID string) []*ExecutionContext {
	lcm.mutex.RLock()
	defer lcm.mutex.RUnlock()

	var contexts []*ExecutionContext
	for _, ctx := range lcm.contexts {
		if ctx.WorkflowID == workflowID {
			contexts = append(contexts, ctx)
		}
	}

	return contexts
}

// cleanup 清理过期的执行上下文
func (lcm *LocalContextManager) cleanup() {
	for {
		select {
		case <-lcm.ticker.C:
			lcm.cleanupExpiredContexts()
		case <-lcm.done:
			return
		}
	}
}

// cleanupExpiredContexts 清理过期上下文
func (lcm *LocalContextManager) cleanupExpiredContexts() {
	lcm.mutex.Lock()
	defer lcm.mutex.Unlock()

	now := time.Now()
	var toRemove []string

	for executionID, ctx := range lcm.contexts {
		// 检查上下文是否过期
		if now.Sub(ctx.createdAt) > lcm.maxAge {
			toRemove = append(toRemove, executionID)
			// 取消上下文
			if ctx.cancel != nil {
				ctx.cancel()
			}
		}
	}

	// 删除过期的上下文
	for _, executionID := range toRemove {
		delete(lcm.contexts, executionID)
	}
}

// Close 关闭上下文管理器
func (lcm *LocalContextManager) Close() {
	lcm.ticker.Stop()
	lcm.done <- true

	// 取消所有活跃的上下文
	lcm.mutex.Lock()
	defer lcm.mutex.Unlock()

	for _, ctx := range lcm.contexts {
		if ctx.cancel != nil {
			ctx.cancel()
		}
	}

	lcm.contexts = make(map[string]*ExecutionContext)
}

// GetStats 获取上下文管理器统计信息
func (lcm *LocalContextManager) GetStats() map[string]any {
	lcm.mutex.RLock()
	defer lcm.mutex.RUnlock()

	return map[string]any{
		"active_contexts":  len(lcm.contexts),
		"cleanup_interval": lcm.cleanupInterval.String(),
		"max_age":          lcm.maxAge.String(),
	}
}

// API基础URL
const API_BASE = '/api/v1';

// 全局状态
let currentState = {
    executions: [],
    currentExecution: null,
    taskResults: [],
    selectedTask: null,
    selectedTaskId: null,
    refreshInterval: null,
    isLoading: false,
    lastUpdateTime: 0,
    userInteracting: false,
    userState: {
        scrollPosition: 0,
        selectedExecutionId: null,
        selectedTaskId: null,
        currentPage: 'list' // 'list' or 'detail'
    },
    loadingStates: {
        executions: false,
        executionDetail: false,
        taskResults: false
    }
};

// 性能优化：防抖函数
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// 状态管理工具函数
function saveUserState() {
    // 保存滚动位置
    if (currentState.userState.currentPage === 'list') {
        currentState.userState.scrollPosition = window.scrollY;
    }
    
    // 保存选中状态
    if (currentState.currentExecution) {
        currentState.userState.selectedExecutionId = currentState.currentExecution.id;
        currentState.userState.currentPage = 'detail';
    } else {
        currentState.userState.currentPage = 'list';
    }
    
    if (currentState.selectedTask) {
        currentState.userState.selectedTaskId = currentState.selectedTask.id;
    }
}

function restoreUserState() {
    // 恢复滚动位置
    if (currentState.userState.currentPage === 'list' && currentState.userState.scrollPosition > 0) {
        setTimeout(() => {
            window.scrollTo(0, currentState.userState.scrollPosition);
        }, 100);
    }
    
    // 恢复选中的任务
    if (currentState.userState.selectedTaskId && currentState.taskResults.length > 0) {
        const selectedTask = currentState.taskResults.find(task => task.id === currentState.userState.selectedTaskId);
        if (selectedTask) {
            setTimeout(() => {
                const taskElements = document.querySelectorAll('.task-item');
                taskElements.forEach((element, index) => {
                    if (currentState.taskResults[index] && currentState.taskResults[index].id === selectedTask.id) {
                        selectTask(selectedTask, element);
                    }
                });
            }, 200);
        }
    }
}

// 数据差异检测
function hasDataChanged(oldData, newData) {
    if (!oldData || !newData) return true;
    if (oldData.length !== newData.length) return true;
    
    for (let i = 0; i < oldData.length; i++) {
        const oldItem = oldData[i];
        const newItem = newData[i];
        
        if (!oldItem || !newItem) return true;
        if (oldItem.id !== newItem.id) return true;
        if (oldItem.status !== newItem.status) return true;
        if (oldItem.started_at !== newItem.started_at) return true;
        if (oldItem.ended_at !== newItem.ended_at) return true;
        if (oldItem.duration !== newItem.duration) return true;
    }
    
    return false;
}

// 任务数据差异检测
function hasTaskDataChanged(oldTasks, newTasks) {
    if (!oldTasks || !newTasks) return true;
    if (oldTasks.length !== newTasks.length) return true;
    
    for (let i = 0; i < oldTasks.length; i++) {
        const oldTask = oldTasks[i];
        const newTask = newTasks[i];
        
        if (!oldTask || !newTask) return true;
        if (oldTask.id !== newTask.id) return true;
        if (oldTask.status !== newTask.status) return true;
        if (oldTask.start_time !== newTask.start_time) return true;
        if (oldTask.end_time !== newTask.end_time) return true;
        if (oldTask.duration !== newTask.duration) return true;
    }
    
    return false;
}

// 页面初始化
document.addEventListener('DOMContentLoaded', function() {
    initializePage();
    setupEventListeners();
});

// 设置事件监听器
function setupEventListeners() {
    // 键盘快捷键
    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape') {
            showExecutionList();
        } else if (e.key === 'F5' || (e.ctrlKey && e.key === 'r')) {
            e.preventDefault();
            refreshData();
        } else if (e.key === ' ' && e.ctrlKey) {
            e.preventDefault();
            toggleAutoRefresh();
        }
    });
    
    // 避免右键菜单干扰
    document.addEventListener('contextmenu', function(e) {
        if (e.target.closest('.task-item') || e.target.closest('.table-row')) {
            e.preventDefault();
        }
    });
    
    // 当用户与页面交互时暂停自动刷新一段时间
    let userInteractionTimer;
    const pauseAutoRefreshOnInteraction = () => {
        clearTimeout(userInteractionTimer);
        currentState.userInteracting = true;
        
        // 2秒后恢复自动刷新
        userInteractionTimer = setTimeout(() => {
            currentState.userInteracting = false;
        }, 2000);
    };
    
    // 监听用户交互
    document.addEventListener('mousedown', pauseAutoRefreshOnInteraction);
    document.addEventListener('keydown', pauseAutoRefreshOnInteraction);
    document.addEventListener('scroll', pauseAutoRefreshOnInteraction);
}

// 切换自动刷新状态
function toggleAutoRefresh() {
    const toggleBtn = document.getElementById('auto-refresh-toggle');
    const toggleIcon = document.getElementById('auto-refresh-icon');
    const toggleText = document.getElementById('auto-refresh-text');
    
    if (currentState.refreshInterval) {
        // 暂停自动刷新
        clearInterval(currentState.refreshInterval);
        currentState.refreshInterval = null;
        
        // 更新按钮外观
        toggleBtn.className = 'btn px-3 py-2 bg-yellow-500 rounded-lg hover:bg-yellow-400 flex items-center text-white text-sm';
        toggleIcon.className = 'fas fa-play mr-1';
        toggleText.textContent = '恢复';
        
        showNotification('自动刷新已暂停 (Ctrl+Space 恢复)', 'warning', 3000);
    } else {
        // 恢复自动刷新
        startAutoRefresh();
        
        // 更新按钮外观
        toggleBtn.className = 'btn px-3 py-2 bg-gray-500 rounded-lg hover:bg-gray-400 flex items-center text-white text-sm';
        toggleIcon.className = 'fas fa-pause mr-1';
        toggleText.textContent = '暂停';
        
        showNotification('自动刷新已恢复', 'success', 2000);
    }
}

// 初始化页面
function initializePage() {
    showLoadingSpinner('executions', true);
    checkHealth();
    loadExecutions();
    
    // 设置定时任务
    setInterval(checkHealth, 30000); // 每30秒检查健康状态
    startAutoRefresh(); // 启动自动刷新
}

// 显示/隐藏加载状态
function showLoadingSpinner(type, show) {
    currentState.loadingStates[type] = show;
    
    switch(type) {
        case 'executions':
            const loadingIndicator = document.getElementById('loading-indicator');
            const skeletonLoader = document.getElementById('skeleton-loader');
            const executionsTable = document.getElementById('executions-table');
            
            if (show) {
                loadingIndicator.classList.remove('hidden');
                if (currentState.executions.length === 0) {
                    skeletonLoader.classList.remove('hidden');
                    executionsTable.style.display = 'none';
                }
            } else {
                loadingIndicator.classList.add('hidden');
                skeletonLoader.classList.add('hidden');
                executionsTable.style.display = 'table-row-group';
            }
            break;
            
        case 'executionDetail':
            const executionInfoSkeleton = document.getElementById('execution-info-skeleton');
            const executionInfo = document.getElementById('execution-info');
            
            if (show) {
                executionInfoSkeleton.classList.remove('hidden');
                executionInfo.classList.add('hidden');
            } else {
                executionInfoSkeleton.classList.add('hidden');
                executionInfo.classList.remove('hidden');
            }
            break;
            
        case 'taskResults':
            const taskLoadingIndicator = document.getElementById('task-loading-indicator');
            const taskListSkeleton = document.getElementById('task-list-skeleton');
            const taskResultsList = document.getElementById('task-results-list');
            
            if (show) {
                taskLoadingIndicator.classList.remove('hidden');
                if (currentState.taskResults.length === 0) {
                    taskListSkeleton.classList.remove('hidden');
                    taskResultsList.style.display = 'none';
                }
            } else {
                taskLoadingIndicator.classList.add('hidden');
                taskListSkeleton.classList.add('hidden');
                taskResultsList.style.display = 'block';
            }
            break;
    }
}

// 健康检查
async function checkHealth() {
    try {
        const response = await fetch(`${API_BASE}/health`);
        const data = await response.json();
        const statusElement = document.getElementById('health-status');
        
        if (response.ok && data.status === 'healthy') {
            statusElement.className = 'px-3 py-1 rounded-full text-sm bg-green-500 flex items-center transition-all duration-300';
            statusElement.innerHTML = '<i class="fas fa-heart mr-1"></i>健康';
        } else {
            statusElement.className = 'px-3 py-1 rounded-full text-sm bg-red-500 flex items-center transition-all duration-300';
            statusElement.innerHTML = '<i class="fas fa-exclamation-triangle mr-1"></i>异常';
        }
    } catch (error) {
        const statusElement = document.getElementById('health-status');
        statusElement.className = 'px-3 py-1 rounded-full text-sm bg-red-500 flex items-center transition-all duration-300';
        statusElement.innerHTML = '<i class="fas fa-times mr-1"></i>离线';
    }
}

// 启动自动刷新（智能刷新）
function startAutoRefresh() {
    if (currentState.refreshInterval) {
        clearInterval(currentState.refreshInterval);
    }
    
    currentState.refreshInterval = setInterval(async () => {
        // 如果用户正在交互，跳过这次刷新
        if (currentState.userInteracting) {
            return;
        }
        
        // 保存用户状态
        saveUserState();
        
        const executionList = document.getElementById('execution-list');
        const executionDetail = document.getElementById('execution-detail');
        
        try {
            if (executionList.style.display !== 'none') {
                // 在列表页面，智能刷新执行列表
                await smartRefreshExecutions();
            } else if (executionDetail.style.display !== 'none' && currentState.currentExecution) {
                // 在详情页面，智能刷新执行详情和任务结果
                await smartRefreshExecutionDetail(currentState.currentExecution.id);
            }
        } catch (error) {
            // 静默处理错误，不打扰用户
            console.log('自动刷新失败:', error);
        }
    }, 5000); // 每5秒刷新一次
}

// 智能刷新执行列表（仅当数据有变化时更新UI）
async function smartRefreshExecutions() {
    try {
        const response = await fetch(`${API_BASE}/executions?limit=50`);
        const newExecutions = await response.json();
        
        // 检查数据是否有变化
        if (!hasDataChanged(currentState.executions, newExecutions || [])) {
            return; // 数据没有变化，不更新UI
        }
        
        const oldExecutions = [...currentState.executions];
        currentState.executions = newExecutions || [];
        
        // 更新执行数量
        document.getElementById('execution-count').textContent = currentState.executions.length;
        
        // 只有数据变化时才重新渲染
        renderExecutionsTable();
        
        // 恢复用户状态
        restoreUserState();
        
        // 仅在有新记录时显示提示
        if (currentState.executions.length > oldExecutions.length) {
            const newCount = currentState.executions.length - oldExecutions.length;
            showNotification(`发现 ${newCount} 条新记录`, 'info', 2000);
        }
        
    } catch (error) {
        console.log('智能刷新执行列表失败:', error);
    }
}

// 智能刷新执行详情（仅当数据有变化时更新UI）
async function smartRefreshExecutionDetail(executionId) {
    try {
        // 检查执行基本信息
        const executionResponse = await fetch(`${API_BASE}/executions/${executionId}`);
        if (!executionResponse.ok) {
            return; // 如果执行不存在，不做任何操作
        }
        
        const newExecution = await executionResponse.json();
        
        // 检查执行基本信息是否有变化
        const executionChanged = !currentState.currentExecution || 
            currentState.currentExecution.status !== newExecution.status ||
            currentState.currentExecution.ended_at !== newExecution.ended_at ||
            currentState.currentExecution.duration !== newExecution.duration;
        
        if (executionChanged) {
            currentState.currentExecution = newExecution;
            renderExecutionInfo(newExecution);
        }
        
        // 检查任务结果
        await smartRefreshTaskResults(executionId);
        
    } catch (error) {
        console.log('智能刷新执行详情失败:', error);
    }
}

// 智能刷新任务结果（仅当数据有变化时更新UI）
async function smartRefreshTaskResults(executionId) {
    try {
        let newTaskResults = [];
        
        const response = await fetch(`${API_BASE}/executions/${executionId}/task-results`);
        if (response.ok) {
            newTaskResults = await response.json() || [];
        } else {
            // 如果API不存在，使用模拟数据
            newTaskResults = generateMockTaskResults(currentState.currentExecution);
        }
        
        // 检查任务数据是否有变化
        if (!hasTaskDataChanged(currentState.taskResults, newTaskResults)) {
            return; // 数据没有变化，不更新UI
        }
        
        currentState.taskResults = newTaskResults;
        
        // 更新任务数量
        document.getElementById('task-count').textContent = currentState.taskResults.length;
        
        // 重新渲染任务列表
        renderTaskResultsList();
        
        // 恢复用户状态
        restoreUserState();
        
    } catch (error) {
        console.log('智能刷新任务结果失败:', error);
    }
}

// 刷新数据
function refreshData() {
    const refreshBtn = document.getElementById('refresh-btn');
    const refreshIcon = document.getElementById('refresh-icon');
    
    // 禁用按钮并显示旋转动画
    refreshBtn.disabled = true;
    refreshIcon.classList.add('fa-spin');
    
    const executionList = document.getElementById('execution-list');
    const executionDetail = document.getElementById('execution-detail');
    
    const refreshPromise = executionList.style.display !== 'none' 
        ? loadExecutions(true)
        : (currentState.currentExecution ? loadExecutionDetail(currentState.currentExecution.id, true) : Promise.resolve());
    
    refreshPromise.finally(() => {
        // 恢复按钮状态
        setTimeout(() => {
            refreshBtn.disabled = false;
            refreshIcon.classList.remove('fa-spin');
            showNotification('数据已刷新', 'success');
        }, 500);
    });
}

// 显示执行列表页面
function showExecutionList() {
    // 保存当前状态
    saveUserState();
    
    document.getElementById('execution-list').style.display = 'block';
    document.getElementById('execution-detail').style.display = 'none';
    document.getElementById('back-button').classList.add('hidden');
    document.getElementById('page-title').textContent = '执行记录监控';
    
    // 更新用户状态
    currentState.userState.currentPage = 'list';
    currentState.userState.selectedExecutionId = null;
    currentState.userState.selectedTaskId = null;
    
    currentState.currentExecution = null;
    currentState.taskResults = [];
    currentState.selectedTask = null;
    loadExecutions();
}

// 显示执行详情页面
function showExecutionDetail(executionId) {
    // 保存当前状态
    saveUserState();
    
    document.getElementById('execution-list').style.display = 'none';
    document.getElementById('execution-detail').style.display = 'block';
    document.getElementById('back-button').classList.remove('hidden');
    
    // 更新用户状态
    currentState.userState.currentPage = 'detail';
    currentState.userState.selectedExecutionId = executionId;
    
    loadExecutionDetail(executionId, true);
}

// 加载执行记录列表
async function loadExecutions(showLoading = true) {
    if (showLoading) {
        showLoadingSpinner('executions', true);
    }
    
    try {
        const response = await fetch(`${API_BASE}/executions?limit=50`);
        const executions = await response.json();
        currentState.executions = executions || [];
        
        // 更新执行数量
        document.getElementById('execution-count').textContent = currentState.executions.length;
        
        renderExecutionsTable();
        
    } catch (error) {
        console.error('加载执行记录失败:', error);
        showNotification('加载执行记录失败', 'error');
    } finally {
        if (showLoading) {
            showLoadingSpinner('executions', false);
        }
    }
}

// 渲染执行记录表格
function renderExecutionsTable() {
    const tbody = document.getElementById('executions-table');
    tbody.innerHTML = '';
    
    if (!currentState.executions || currentState.executions.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="7" class="px-6 py-12 text-center text-gray-500">
                    <div class="flex flex-col items-center space-y-3">
                        <i class="fas fa-database text-4xl text-gray-300"></i>
                        <div class="text-lg font-medium">暂无执行记录</div>
                        <div class="text-sm text-gray-400">当有新的工作流执行时，记录会显示在这里</div>
                    </div>
                </td>
            </tr>
        `;
        return;
    }
    
    currentState.executions.forEach((execution, index) => {
        const row = document.createElement('tr');
        row.className = 'table-row hover:bg-gray-50 cursor-pointer transition-all duration-150';
        row.setAttribute('tabindex', '0'); // 支持键盘导航
        
        row.innerHTML = `
            <td class="px-6 py-4 whitespace-nowrap">
                <div class="font-mono text-sm text-gray-900 font-medium">${execution.id.substring(0, 12)}...</div>
                <div class="text-xs text-gray-500 truncate max-w-32" title="${execution.id}">${execution.id}</div>
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm font-medium text-gray-900 truncate max-w-40" title="${execution.workflow_id}">${execution.workflow_id}</div>
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
                <span class="px-3 py-1 inline-flex items-center text-xs leading-5 font-semibold rounded-full ${getStatusClass(execution.status)} transition-all duration-200">
                    <i class="fas ${getStatusIcon(execution.status)} mr-1.5 flex-shrink-0"></i>
                    <span>${getStatusText(execution.status)}</span>
                </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                ${execution.started_at ? formatDateTime(execution.started_at) : '-'}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                ${execution.duration ? formatDuration(execution.duration) : '-'}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                <span class="px-2 py-1 bg-gray-100 rounded-full text-xs">${execution.retry_count || 0}</span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                <button onclick="showExecutionDetail('${execution.id}')" 
                        class="btn text-blue-600 hover:text-blue-900 px-3 py-1 rounded-md hover:bg-blue-50 transition-all duration-150">
                    <i class="fas fa-eye mr-1"></i>详情
                </button>
            </td>
        `;
        
        // 添加点击事件
        row.addEventListener('click', (e) => {
            if (!e.target.closest('button')) {
                showExecutionDetail(execution.id);
            }
        });
        
        // 添加键盘事件
        row.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                showExecutionDetail(execution.id);
            }
        });
        
        tbody.appendChild(row);
    });
}

// 加载执行详情
async function loadExecutionDetail(executionId, showLoading = true) {
    if (showLoading) {
        showLoadingSpinner('executionDetail', true);
    }
    
    try {
        // 加载执行基本信息
        const response = await fetch(`${API_BASE}/executions/${executionId}`);
        if (!response.ok) {
            throw new Error('执行记录不存在');
        }
        
        const execution = await response.json();
        currentState.currentExecution = execution;
        
        document.getElementById('page-title').textContent = `执行详情 - ${execution.id.substring(0, 12)}...`;
        
        renderExecutionInfo(execution);
        await loadTaskResults(executionId, showLoading);
        
    } catch (error) {
        console.error('加载执行详情失败:', error);
        showNotification('加载执行详情失败', 'error');
        showExecutionList();
    } finally {
        if (showLoading) {
            showLoadingSpinner('executionDetail', false);
        }
    }
}

// 渲染执行信息
function renderExecutionInfo(execution) {
    const container = document.getElementById('execution-info');
    
    container.innerHTML = `
        <div class="info-item bg-gray-50 p-3 rounded-lg">
            <label class="block text-xs font-medium text-gray-600 mb-1 uppercase tracking-wide">执行ID</label>
            <div class="text-sm text-gray-900 font-mono break-all">${execution.id}</div>
        </div>
        <div class="info-item bg-gray-50 p-3 rounded-lg">
            <label class="block text-xs font-medium text-gray-600 mb-1 uppercase tracking-wide">工作流ID</label>
            <div class="text-sm text-gray-900 truncate" title="${execution.workflow_id}">${execution.workflow_id}</div>
        </div>
        <div class="info-item bg-gray-50 p-3 rounded-lg">
            <label class="block text-xs font-medium text-gray-600 mb-1 uppercase tracking-wide">状态</label>
            <span class="px-2 py-1 inline-flex items-center text-xs leading-4 font-semibold rounded-full ${getStatusClass(execution.status)}">
                <i class="fas ${getStatusIcon(execution.status)} mr-1.5 flex-shrink-0"></i>
                <span>${getStatusText(execution.status)}</span>
            </span>
        </div>
        <div class="info-item bg-gray-50 p-3 rounded-lg">
            <label class="block text-xs font-medium text-gray-600 mb-1 uppercase tracking-wide">开始时间</label>
            <div class="text-sm text-gray-900">${execution.started_at ? formatDateTime(execution.started_at) : '-'}</div>
        </div>
        <div class="info-item bg-gray-50 p-3 rounded-lg">
            <label class="block text-xs font-medium text-gray-600 mb-1 uppercase tracking-wide">结束时间</label>
            <div class="text-sm text-gray-900">${execution.ended_at ? formatDateTime(execution.ended_at) : '未结束'}</div>
        </div>
        <div class="info-item bg-gray-50 p-3 rounded-lg">
            <label class="block text-xs font-medium text-gray-600 mb-1 uppercase tracking-wide">耗时</label>
            <div class="text-sm text-gray-900 font-medium">${execution.duration ? formatDuration(execution.duration) : '-'}</div>
        </div>
    `;
}

// 加载任务结果
async function loadTaskResults(executionId, showLoading = true) {
    if (showLoading) {
        showLoadingSpinner('taskResults', true);
    }
    
    try {
        const response = await fetch(`${API_BASE}/executions/${executionId}/task-results`);
        if (response.ok) {
            const taskResults = await response.json();
            currentState.taskResults = taskResults || [];
        } else {
            // 如果API不存在，使用模拟数据
            currentState.taskResults = generateMockTaskResults(currentState.currentExecution);
        }
        
        // 更新任务数量
        document.getElementById('task-count').textContent = currentState.taskResults.length;
        
        renderTaskResultsList();
        clearTaskDetail(); // 清空右侧详情
        
    } catch (error) {
        console.error('加载任务结果失败:', error);
        // 使用模拟数据
        currentState.taskResults = generateMockTaskResults(currentState.currentExecution);
        document.getElementById('task-count').textContent = currentState.taskResults.length;
        renderTaskResultsList();
        clearTaskDetail();
    } finally {
        if (showLoading) {
            showLoadingSpinner('taskResults', false);
        }
    }
}

// 生成模拟任务结果（保持原有逻辑）
function generateMockTaskResults(execution) {
    const taskNames = ['extract', 'validate', 'transform', 'load'];
    const tasks = [];
    
    taskNames.forEach((taskId, index) => {
        let status = 'pending';
        let startTime = null;
        let endTime = null;
        let duration = null;
        let error = null;
        
        if (execution.status === 'success') {
            status = 'success';
            startTime = new Date(new Date(execution.started_at).getTime() + index * 2000);
            endTime = new Date(startTime.getTime() + Math.floor(Math.random() * 5000) + 1000);
            duration = endTime - startTime;
        } else if (execution.status === 'running') {
            if (index === 0) {
                status = 'success';
                startTime = new Date(execution.started_at);
                endTime = new Date(startTime.getTime() + 2000);
                duration = 2000;
            } else if (index === 1) {
                status = 'running';
                startTime = new Date(new Date(execution.started_at).getTime() + 2000);
            }
        } else if (execution.status === 'failed') {
            if (index < 2) {
                status = 'success';
                startTime = new Date(new Date(execution.started_at).getTime() + index * 2000);
                endTime = new Date(startTime.getTime() + Math.floor(Math.random() * 3000) + 1000);
                duration = endTime - startTime;
            } else if (index === 2) {
                status = 'failed';
                startTime = new Date(new Date(execution.started_at).getTime() + 4000);
                endTime = new Date(startTime.getTime() + 1500);
                duration = 1500;
                error = '数据验证失败：格式不正确';
            }
        }
        
        tasks.push({
            id: `${execution.id}_${taskId}`,
            execution_id: execution.id,
            task_id: taskId,
            status: status,
            input: generateMockTaskInput(taskId),
            output: status === 'success' ? generateMockTaskOutput(taskId) : null,
            error: error,
            start_time: startTime ? startTime.toISOString() : null,
            end_time: endTime ? endTime.toISOString() : null,
            duration: duration,
            retry_count: 0
        });
    });
    
    return tasks;
}

// 生成模拟任务输入
function generateMockTaskInput(taskId) {
    const inputs = {
        'extract': { source: 'database', table: 'users', batch_size: 1000 },
        'validate': { rules: ['email', 'phone', 'age'], strict: true, threshold: 0.95 },
        'transform': { format: 'json', encoding: 'utf8', normalize: true },
        'load': { target: 'warehouse', batch_size: 500, mode: 'append' }
    };
    return inputs[taskId] || {};
}

// 生成模拟任务输出
function generateMockTaskOutput(taskId) {
    const outputs = {
        'extract': { 
            records: 1000, 
            extracted_at: new Date().toISOString(),
            source_schema: 'users_v2',
            total_size: '2.3MB'
        },
        'validate': { 
            total_records: 1000,
            valid_records: 950, 
            invalid_records: 50,
            validation_rate: 0.95,
            errors: ['invalid_email', 'missing_phone']
        },
        'transform': { 
            input_records: 950,
            transformed_records: 950, 
            format: 'json',
            transformations: ['normalize_names', 'standardize_phone']
        },
        'load': { 
            loaded_records: 950, 
            target_table: 'processed_data',
            load_mode: 'append',
            index_updated: true
        }
    };
    return outputs[taskId] || {};
}

// 渲染任务结果列表
function renderTaskResultsList() {
    const container = document.getElementById('task-results-list');
    container.innerHTML = '';
    
    if (!currentState.taskResults || currentState.taskResults.length === 0) {
        container.innerHTML = `
            <div class="text-center text-gray-500 py-8">
                <i class="fas fa-cogs text-3xl text-gray-300 mb-3"></i>
                <div class="text-sm font-medium">暂无任务数据</div>
                <div class="text-xs text-gray-400 mt-1">任务执行后会显示在这里</div>
            </div>
        `;
        return;
    }
    
    currentState.taskResults.forEach((task, index) => {
        const taskItem = document.createElement('div');
        taskItem.className = 'task-item p-4 border-2 border-transparent rounded-lg cursor-pointer';
        taskItem.setAttribute('tabindex', '0');
        
        taskItem.innerHTML = `
            <div class="flex justify-between items-start mb-3">
                <div class="flex items-center space-x-2">
                    <div class="font-medium text-gray-900">${getTaskDisplayName(task.task_id)}</div>
                    <div class="text-xs text-gray-500 font-mono bg-gray-100 px-2 py-1 rounded">${task.task_id}</div>
                </div>
                <span class="px-2 py-1 inline-flex items-center text-xs rounded-full ${getStatusClass(task.status)}">
                    <i class="fas ${getStatusIcon(task.status)} mr-1.5 flex-shrink-0"></i>
                    <span>${getStatusText(task.status)}</span>
                </span>
            </div>
            <div class="grid grid-cols-2 gap-2 text-xs text-gray-500">
                <div>
                    <i class="fas fa-clock mr-1"></i>
                    ${task.start_time ? formatDateTime(task.start_time) : '未开始'}
                </div>
                ${task.duration ? `<div><i class="fas fa-stopwatch mr-1"></i>${formatDuration(task.duration)}</div>` : '<div></div>'}
            </div>
        `;
        
        // 添加点击事件
        taskItem.addEventListener('click', () => selectTask(task, taskItem));
        
        // 添加键盘事件
        taskItem.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                selectTask(task, taskItem);
            }
        });
        
        container.appendChild(taskItem);
    });
}

// 选择任务（改进的版本）
function selectTask(task, taskElement) {
    // 移除之前的选中状态
    document.querySelectorAll('.task-item').forEach(item => {
        item.classList.remove('selected');
    });
    
    // 设置当前选中状态
    taskElement.classList.add('selected');
    currentState.selectedTask = task;
    
    // 保存选中状态
    currentState.userState.selectedTaskId = task.id;
    saveUserState();
    
    // 渲染任务详情
    renderTaskDetail(task);
    
    // 滚动到详情区域（移动端友好）
    if (window.innerWidth < 768) {
        document.getElementById('task-detail-content').scrollIntoView({ 
            behavior: 'smooth', 
            block: 'start' 
        });
    }
}

// 渲染任务详情（优化版本）
function renderTaskDetail(task) {
    const container = document.getElementById('task-detail-content');
    
    container.innerHTML = `
        <div class="space-y-4 animate__animated animate__fadeIn">
            <!-- 任务基本信息 -->
            <div class="bg-gradient-to-r from-blue-50 to-indigo-50 rounded-lg p-4 border border-blue-100">
                <h4 class="text-sm font-semibold text-gray-800 mb-3 flex items-center">
                    <i class="fas fa-info-circle mr-2 text-blue-600"></i>
                    基本信息
                </h4>
                <div class="grid grid-cols-1 gap-3 text-sm">
                    <div class="flex justify-between">
                        <span class="font-medium text-gray-700">任务名称:</span>
                        <span class="text-gray-900">${getTaskDisplayName(task.task_id)}</span>
                    </div>
                    <div class="flex justify-between">
                        <span class="font-medium text-gray-700">任务ID:</span>
                        <span class="text-gray-900 font-mono text-xs">${task.task_id}</span>
                    </div>
                    <div class="flex justify-between">
                        <span class="font-medium text-gray-700">状态:</span>
                        <span class="px-2 py-1 inline-flex items-center text-xs rounded-full ${getStatusClass(task.status)}">
                            <i class="fas ${getStatusIcon(task.status)} mr-1.5 flex-shrink-0"></i>
                            <span>${getStatusText(task.status)}</span>
                        </span>
                    </div>
                    <div class="flex justify-between">
                        <span class="font-medium text-gray-700">开始时间:</span>
                        <span class="text-gray-900">${task.start_time ? formatDateTime(task.start_time) : '-'}</span>
                    </div>
                    <div class="flex justify-between">
                        <span class="font-medium text-gray-700">结束时间:</span>
                        <span class="text-gray-900">${task.end_time ? formatDateTime(task.end_time) : '-'}</span>
                    </div>
                    <div class="flex justify-between">
                        <span class="font-medium text-gray-700">耗时:</span>
                        <span class="text-gray-900 font-medium">${task.duration ? formatDuration(task.duration) : '-'}</span>
                    </div>
                    <div class="flex justify-between">
                        <span class="font-medium text-gray-700">重试次数:</span>
                        <span class="px-2 py-1 bg-gray-100 text-gray-700 rounded text-xs">${task.retry_count || 0}</span>
                    </div>
                </div>
            </div>
            
            <!-- 输入数据 -->
            <div class="bg-gradient-to-r from-green-50 to-emerald-50 rounded-lg p-4 border border-green-100">
                <h4 class="text-sm font-semibold text-gray-800 mb-3 flex items-center">
                    <i class="fas fa-sign-in-alt mr-2 text-green-600"></i>
                    输入数据
                </h4>
                <div class="bg-white rounded border p-3 max-h-32 overflow-auto">
                    <pre class="text-xs text-gray-800 whitespace-pre-wrap">${JSON.stringify(task.input, null, 2)}</pre>
                </div>
            </div>
            
            <!-- 输出数据 -->
            ${task.output ? `
                <div class="bg-gradient-to-r from-purple-50 to-violet-50 rounded-lg p-4 border border-purple-100">
                    <h4 class="text-sm font-semibold text-gray-800 mb-3 flex items-center">
                        <i class="fas fa-sign-out-alt mr-2 text-purple-600"></i>
                        输出数据
                    </h4>
                    <div class="bg-white rounded border p-3 max-h-32 overflow-auto">
                        <pre class="text-xs text-gray-800 whitespace-pre-wrap">${JSON.stringify(task.output, null, 2)}</pre>
                    </div>
                </div>
            ` : ''}
            
            <!-- 错误信息 -->
            ${task.error ? `
                <div class="bg-gradient-to-r from-red-50 to-pink-50 rounded-lg p-4 border border-red-100">
                    <h4 class="text-sm font-semibold text-gray-800 mb-3 flex items-center">
                        <i class="fas fa-exclamation-triangle mr-2 text-red-600"></i>
                        错误信息
                    </h4>
                    <div class="text-sm text-red-800 bg-white p-3 rounded border">${task.error}</div>
                </div>
            ` : ''}
            
            <!-- 日志信息 -->
            <div class="bg-gradient-to-r from-yellow-50 to-amber-50 rounded-lg p-4 border border-yellow-100">
                <h4 class="text-sm font-semibold text-gray-800 mb-3 flex items-center">
                    <i class="fas fa-file-alt mr-2 text-yellow-600"></i>
                    执行日志
                </h4>
                <div class="space-y-2 max-h-40 overflow-y-auto">
                    ${generateTaskLogs(task).map(log => `
                        <div class="text-xs bg-white p-3 rounded border">
                            <div class="flex justify-between items-center mb-1">
                                <span class="font-medium text-gray-500">${formatDateTime(log.timestamp)}</span>
                                <span class="px-2 py-1 text-xs rounded ${getLogLevelClass(log.level)}">${log.level.toUpperCase()}</span>
                            </div>
                            <div class="text-gray-800">${log.message}</div>
                        </div>
                    `).join('')}
                </div>
            </div>
        </div>
    `;
}

// 生成任务日志（保持原有逻辑）
function generateTaskLogs(task) {
    const logs = [];
    const baseTime = task.start_time ? new Date(task.start_time) : new Date();
    
    if (task.status === 'pending') {
        return logs;
    }
    
    logs.push({
        timestamp: baseTime,
        level: 'info',
        message: `任务 ${getTaskDisplayName(task.task_id)} 开始执行`
    });
    
    if (task.status === 'running') {
        logs.push({
            timestamp: new Date(baseTime.getTime() + 500),
            level: 'info',
            message: '正在处理输入数据...'
        });
        logs.push({
            timestamp: new Date(baseTime.getTime() + 1000),
            level: 'debug',
            message: `输入参数：${JSON.stringify(task.input)}`
        });
    } else if (task.status === 'success') {
        logs.push({
            timestamp: new Date(baseTime.getTime() + 200),
            level: 'debug',
            message: `输入参数：${JSON.stringify(task.input)}`
        });
        logs.push({
            timestamp: new Date(baseTime.getTime() + 500),
            level: 'info',
            message: '数据处理中...'
        });
        logs.push({
            timestamp: new Date(baseTime.getTime() + (task.duration || 1000) - 100),
            level: 'info',
            message: `输出结果：${JSON.stringify(task.output)}`
        });
        logs.push({
            timestamp: new Date(baseTime.getTime() + (task.duration || 1000)),
            level: 'info',
            message: `任务 ${getTaskDisplayName(task.task_id)} 执行成功`
        });
    } else if (task.status === 'failed') {
        logs.push({
            timestamp: new Date(baseTime.getTime() + 200),
            level: 'debug',
            message: `输入参数：${JSON.stringify(task.input)}`
        });
        logs.push({
            timestamp: new Date(baseTime.getTime() + 500),
            level: 'warn',
            message: '检测到数据异常'
        });
        logs.push({
            timestamp: new Date(baseTime.getTime() + (task.duration || 1000)),
            level: 'error',
            message: task.error || '任务执行失败'
        });
    }
    
    return logs;
}

// 清空任务详情
function clearTaskDetail() {
    const container = document.getElementById('task-detail-content');
    container.innerHTML = `
        <div class="text-center text-gray-500 py-12">
            <i class="fas fa-mouse-pointer text-3xl text-gray-300 mb-3"></i>
            <div class="text-sm">请在左侧选择一个任务查看详情</div>
            <div class="text-xs text-gray-400 mt-2">点击任务卡片查看输入输出和日志</div>
        </div>
    `;
    
    // 清除选中状态
    currentState.selectedTask = null;
    currentState.userState.selectedTaskId = null;
    
    // 移除所有任务项的选中样式
    document.querySelectorAll('.task-item').forEach(item => {
        item.classList.remove('selected');
    });
}

// 工具函数
function getStatusClass(status) {
    switch (status) {
        case 'running': return 'bg-yellow-100 text-yellow-800 border-yellow-200';
        case 'success': return 'bg-green-100 text-green-800 border-green-200';
        case 'failed': return 'bg-red-100 text-red-800 border-red-200';
        case 'pending': return 'bg-gray-100 text-gray-800 border-gray-200';
        case 'cancelled': return 'bg-purple-100 text-purple-800 border-purple-200';
        default: return 'bg-gray-100 text-gray-800 border-gray-200';
    }
}

function getStatusIcon(status) {
    switch (status) {
        case 'running': return 'fa-spinner fa-spin';
        case 'success': return 'fa-check-circle';
        case 'failed': return 'fa-times-circle';
        case 'pending': return 'fa-clock';
        case 'cancelled': return 'fa-stop-circle';
        default: return 'fa-question-circle';
    }
}

function getStatusText(status) {
    switch (status) {
        case 'running': return '运行中';
        case 'success': return '成功';
        case 'failed': return '失败';
        case 'pending': return '等待中';
        case 'cancelled': return '已取消';
        default: return status;
    }
}

function getTaskDisplayName(taskId) {
    const names = {
        'extract': '数据提取',
        'validate': '数据验证', 
        'transform': '数据转换',
        'load': '数据加载'
    };
    return names[taskId] || taskId;
}

function formatDateTime(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
}

function formatDuration(milliseconds) {
    if (milliseconds < 1000) {
        return `${milliseconds}ms`;
    } else if (milliseconds < 60000) {
        return `${(milliseconds / 1000).toFixed(1)}s`;
    } else {
        const minutes = Math.floor(milliseconds / 60000);
        const seconds = Math.floor((milliseconds % 60000) / 1000);
        return `${minutes}m ${seconds}s`;
    }
}

function getLogLevelClass(level) {
    switch (level.toLowerCase()) {
        case 'debug': return 'bg-gray-100 text-gray-700';
        case 'info': return 'bg-blue-100 text-blue-700';
        case 'warn': return 'bg-yellow-100 text-yellow-700';
        case 'error': return 'bg-red-100 text-red-700';
        default: return 'bg-gray-100 text-gray-700';
    }
}

// 通知系统（改进版本）
function showNotification(message, type = 'info', duration = 3000) {
    const container = document.getElementById('notification-container');
    const notification = document.createElement('div');
    
    const colors = {
        success: 'bg-green-500',
        error: 'bg-red-500',
        info: 'bg-blue-500',
        warning: 'bg-yellow-500'
    };
    
    const icons = {
        success: 'fa-check-circle',
        error: 'fa-times-circle',
        info: 'fa-info-circle',
        warning: 'fa-exclamation-triangle'
    };
    
    notification.className = `notification ${colors[type] || colors.info} text-white px-4 py-3 rounded-lg shadow-lg flex items-center space-x-3 min-w-72 max-w-sm`;
    notification.innerHTML = `
        <i class="fas ${icons[type] || icons.info} flex-shrink-0"></i>
        <span class="flex-1 text-sm font-medium">${message}</span>
        <button onclick="removeNotification(this)" class="text-white hover:text-gray-200 transition-colors ml-2 flex-shrink-0">
            <i class="fas fa-times"></i>
        </button>
    `;
    
    container.appendChild(notification);
    
    // 自动移除
    setTimeout(() => {
        removeNotification(notification.querySelector('button'));
    }, duration);
}

function removeNotification(button) {
    const notification = button.closest('.notification');
    if (notification) {
        notification.classList.add('removing');
        setTimeout(() => {
            if (notification.parentElement) {
                notification.remove();
            }
        }, 300);
    }
}
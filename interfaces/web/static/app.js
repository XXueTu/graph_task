// API基础URL
const API_BASE = '/api/v1';

// 全局状态
let currentData = {
    workflows: [],
    executions: [],
    retries: []
};

// 页面初始化
document.addEventListener('DOMContentLoaded', function() {
    checkHealth();
    refreshData();
    setInterval(checkHealth, 30000); // 每30秒检查健康状态
    setInterval(autoRefreshExecutions, 5000); // 每5秒自动刷新执行状态
});

// 健康检查
async function checkHealth() {
    try {
        const response = await fetch(`${API_BASE}/health`);
        const data = await response.json();
        const statusElement = document.getElementById('health-status');
        
        if (response.ok && data.status === 'healthy') {
            statusElement.className = 'px-3 py-1 rounded-full text-sm bg-green-500';
            statusElement.innerHTML = '<i class="fas fa-heart mr-1"></i>健康';
        } else {
            statusElement.className = 'px-3 py-1 rounded-full text-sm bg-red-500';
            statusElement.innerHTML = '<i class="fas fa-exclamation-triangle mr-1"></i>异常';
        }
    } catch (error) {
        const statusElement = document.getElementById('health-status');
        statusElement.className = 'px-3 py-1 rounded-full text-sm bg-red-500';
        statusElement.innerHTML = '<i class="fas fa-times mr-1"></i>离线';
    }
}

// 刷新所有数据
function refreshData() {
    loadWorkflows();
    loadExecutions();
    loadRetries();
    loadSDKExample();
}

// 自动刷新执行状态
function autoRefreshExecutions() {
    if (document.getElementById('executions-content').style.display !== 'none') {
        loadExecutions();
    }
}

// 标签页切换
function showTab(tabName) {
    // 隐藏所有标签页内容
    const contents = document.querySelectorAll('.tab-content');
    contents.forEach(content => content.classList.add('hidden'));
    
    // 移除所有标签页的active状态
    const tabs = document.querySelectorAll('.tab-button');
    tabs.forEach(tab => {
        tab.classList.remove('active-tab');
        tab.classList.add('text-gray-500', 'hover:text-gray-700');
    });
    
    // 显示选中的标签页内容
    document.getElementById(`${tabName}-content`).classList.remove('hidden');
    
    // 设置选中标签页的active状态
    const activeTab = document.getElementById(`${tabName}-tab`);
    activeTab.classList.add('active-tab');
    activeTab.classList.remove('text-gray-500', 'hover:text-gray-700');
    
    // 根据标签页加载相应数据
    switch(tabName) {
        case 'workflows':
            loadWorkflows();
            break;
        case 'executions':
            loadExecutions();
            break;
        case 'retries':
            loadRetries();
            break;
        case 'sdk':
            loadSDKExample();
            break;
    }
}

// 加载工作流列表
async function loadWorkflows() {
    try {
        const response = await fetch(`${API_BASE}/workflows`);
        const workflows = await response.json();
        currentData.workflows = workflows;
        
        const tbody = document.getElementById('workflows-list');
        tbody.innerHTML = '';
        
        if (!workflows || workflows.length === 0) {
            tbody.innerHTML = `
                <tr>
                    <td colspan="6" class="px-6 py-4 text-center text-gray-500">
                        暂无工作流。请使用SDK注册工作流，或创建工作流模板。
                    </td>
                </tr>
            `;
            return;
        }
        
        workflows.forEach(workflow => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td class="px-6 py-4 whitespace-nowrap">
                    <div class="font-medium text-gray-900">${workflow.name}</div>
                    <div class="text-sm text-gray-500">${workflow.id}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${workflow.version}</td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full 
                                 ${getStatusClass(workflow.status)}">
                        ${getStatusText(workflow.status)}
                    </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${workflow.task_count}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    ${formatDateTime(workflow.created_at)}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm font-medium space-x-2">
                    <button onclick="viewWorkflowDetails('${workflow.id}')" 
                            class="text-blue-600 hover:text-blue-900">查看</button>
                    <button onclick="executeWorkflowById('${workflow.id}')" 
                            class="text-green-600 hover:text-green-900">执行</button>
                    <button onclick="deleteWorkflow('${workflow.id}')" 
                            class="text-red-600 hover:text-red-900">删除</button>
                </td>
            `;
            tbody.appendChild(row);
        });
        
        // 更新执行模态框中的工作流选择器
        updateWorkflowSelector();
        
    } catch (error) {
        console.error('加载工作流失败:', error);
        showErrorMessage('加载工作流失败');
    }
}

// 加载执行记录
async function loadExecutions() {
    try {
        const response = await fetch(`${API_BASE}/executions`);
        const executions = await response.json();
        currentData.executions = executions;
        
        const tbody = document.getElementById('executions-list');
        tbody.innerHTML = '';
        
        if (!executions || executions.length === 0) {
            tbody.innerHTML = `
                <tr>
                    <td colspan="8" class="px-6 py-4 text-center text-gray-500">
                        暂无执行记录
                    </td>
                </tr>
            `;
            return;
        }
        
        for (const execution of executions) {
            // 获取执行状态
            let statusInfo = null;
            try {
                const statusResponse = await fetch(`${API_BASE}/executions/${execution.id}/status`);
                if (statusResponse.ok) {
                    statusInfo = await statusResponse.json();
                }
            } catch (e) {
                // 忽略状态获取错误
            }
            
            const row = document.createElement('tr');
            row.innerHTML = `
                <td class="px-6 py-4 whitespace-nowrap">
                    <div class="font-medium text-gray-900">${execution.id.substring(0, 8)}...</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${execution.workflow_id}</td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full 
                                 ${getStatusClass(execution.status)}">
                        ${getStatusText(execution.status)}
                    </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                    ${statusInfo ? `
                        <div class="w-full bg-gray-200 rounded-full h-2">
                            <div class="bg-blue-600 h-2 rounded-full" style="width: ${statusInfo.progress}%"></div>
                        </div>
                        <div class="text-xs text-gray-500 mt-1">${statusInfo.progress.toFixed(1)}%</div>
                    ` : '-'}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    ${formatDateTime(execution.started_at)}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    ${execution.duration ? formatDuration(execution.duration) : '-'}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${execution.retry_count || 0}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm font-medium space-x-2">
                    <button onclick="viewExecutionDetails('${execution.id}')" 
                            class="text-blue-600 hover:text-blue-900">查看</button>
                    <button onclick="viewExecutionLogs('${execution.id}')" 
                            class="text-green-600 hover:text-green-900">日志</button>
                    ${execution.status === 'running' ? `
                        <button onclick="cancelExecution('${execution.id}')" 
                                class="text-red-600 hover:text-red-900">取消</button>
                    ` : ''}
                </td>
            `;
            tbody.appendChild(row);
        }
        
    } catch (error) {
        console.error('加载执行记录失败:', error);
        showErrorMessage('加载执行记录失败');
    }
}

// 加载重试信息
async function loadRetries() {
    try {
        const response = await fetch(`${API_BASE}/retries`);
        const retries = await response.json();
        currentData.retries = retries;
        
        const tbody = document.getElementById('retries-list');
        tbody.innerHTML = '';
        
        if (!retries || retries.length === 0) {
            tbody.innerHTML = `
                <tr>
                    <td colspan="7" class="px-6 py-4 text-center text-gray-500">
                        暂无需要重试的执行
                    </td>
                </tr>
            `;
            return;
        }
        
        retries.forEach(retry => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td class="px-6 py-4 whitespace-nowrap">
                    <div class="font-medium text-gray-900">${retry.execution_id.substring(0, 8)}...</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${retry.workflow_id}</td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <div class="text-sm text-gray-900 max-w-xs truncate" title="${retry.failure_reason}">
                        ${retry.failure_reason}
                    </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${retry.retry_count}</td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full 
                                 ${getStatusClass(retry.status)}">
                        ${getStatusText(retry.status)}
                    </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    ${formatDateTime(retry.last_retry_at)}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm font-medium space-x-2">
                    <button onclick="viewRetryDetails('${retry.execution_id}')" 
                            class="text-blue-600 hover:text-blue-900">查看</button>
                    ${retry.status !== 'exhausted' ? `
                        <button onclick="manualRetry('${retry.execution_id}')" 
                                class="text-green-600 hover:text-green-900">重试</button>
                    ` : ''}
                </td>
            `;
            tbody.appendChild(row);
        });
        
    } catch (error) {
        console.error('加载重试信息失败:', error);
        showErrorMessage('加载重试信息失败');
    }
}

// 加载SDK示例
function loadSDKExample() {
    const exampleCode = `package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/XXueTu/graph_task"
    "github.com/XXueTu/graph_task/domain/workflow"
)

func main() {
    // 1. 创建引擎配置
    config := graph_task.DefaultEngineConfig()
    config.MySQLDSN = "user:pass@tcp(localhost:3306)/taskdb?charset=utf8mb4&parseTime=True&loc=Local"
    config.WebPort = 8080 // 启用Web界面
    
    // 2. 创建引擎
    engine, err := graph_task.NewEngine(config)
    if err != nil {
        log.Fatal(err)
    }
    defer engine.Close()
    
    // 3. 创建工作流
    builder := engine.CreateWorkflow("my-workflow")
    workflow, err := builder.
        SetDescription("我的第一个工作流").
        SetVersion("1.0.0").
        AddTask("preprocess", "数据预处理", preprocessHandler).
        AddTask("process", "数据处理", processHandler).
        AddTask("postprocess", "数据后处理", postprocessHandler).
        AddDependency("preprocess", "process").
        AddDependency("process", "postprocess").
        Build()
    
    if err != nil {
        log.Fatal(err)
    }
    
    // 4. 发布工作流
    if err := engine.PublishWorkflow(workflow); err != nil {
        log.Fatal(err)
    }
    
    // 5. 启动Web服务器 (在goroutine中启动)
    go func() {
        if err := engine.StartWebServer(); err != nil {
            log.Printf("Web服务器启动失败: %v", err)
        }
    }()
    
    // 6. 执行工作流
    ctx := context.Background()
    input := map[string]interface{}{
        "data": []int{1, 2, 3, 4, 5},
        "factor": 2,
    }
    
    execution, err := engine.Execute(ctx, "my-workflow", input)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("执行完成: %s\\n", execution.ID())
    fmt.Printf("状态: %s\\n", execution.Status())
    fmt.Printf("结果: %+v\\n", execution.Output())
    
    // 7. 保持程序运行以提供Web服务
    select {}
}

// 任务处理器示例
func preprocessHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
    data := input["data"].([]int)
    
    // 预处理：过滤负数
    var filtered []int
    for _, v := range data {
        if v >= 0 {
            filtered = append(filtered, v)
        }
    }
    
    return map[string]interface{}{
        "filtered_data": filtered,
        "original_count": len(data),
    }, nil
}

func processHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
    // 获取上一个任务的输出
    preprocessOutput := input["preprocess_output"].(map[string]interface{})
    data := preprocessOutput["filtered_data"].([]int)
    factor := input["factor"].(int)
    
    // 处理：乘以因子
    var result []int
    sum := 0
    for _, v := range data {
        processed := v * factor
        result = append(result, processed)
        sum += processed
    }
    
    return map[string]interface{}{
        "result": result,
        "sum": sum,
        "average": float64(sum) / float64(len(result)),
    }, nil
}

func postprocessHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
    // 获取处理结果
    processOutput := input["process_output"].(map[string]interface{})
    result := processOutput["result"].([]int)
    
    // 后处理：生成统计信息
    if len(result) == 0 {
        return map[string]interface{}{
            "statistics": map[string]interface{}{
                "count": 0,
            },
        }, nil
    }
    
    min, max := result[0], result[0]
    for _, v := range result {
        if v < min {
            min = v
        }
        if v > max {
            max = v
        }
    }
    
    return map[string]interface{}{
        "final_result": result,
        "statistics": map[string]interface{}{
            "min": min,
            "max": max,
            "count": len(result),
        },
        "status": "completed",
    }, nil
}`;

    document.getElementById('sdk-example-code').textContent = exampleCode;
}

// 更新工作流选择器
function updateWorkflowSelector() {
    const selector = document.getElementById('execute-workflow-id');
    const filterSelector = document.getElementById('workflow-filter');
    
    // 清空现有选项
    selector.innerHTML = '<option value="">选择工作流</option>';
    filterSelector.innerHTML = '<option value="">所有工作流</option>';
    
    currentData.workflows.forEach(workflow => {
        if (workflow.status === 'published') {
            const option = document.createElement('option');
            option.value = workflow.id;
            option.textContent = `${workflow.name} (${workflow.version})`;
            selector.appendChild(option);
            
            const filterOption = document.createElement('option');
            filterOption.value = workflow.id;
            filterOption.textContent = workflow.name;
            filterSelector.appendChild(filterOption);
        }
    });
}

// 显示错误消息
function showErrorMessage(message) {
    // 创建错误提示
    const alert = document.createElement('div');
    alert.className = 'fixed top-4 right-4 bg-red-500 text-white px-4 py-2 rounded shadow-lg z-50';
    alert.innerHTML = `
        <div class="flex items-center">
            <i class="fas fa-exclamation-circle mr-2"></i>
            <span>${message}</span>
            <button onclick="this.parentElement.parentElement.remove()" class="ml-4 text-white hover:text-gray-200">
                <i class="fas fa-times"></i>
            </button>
        </div>
    `;
    document.body.appendChild(alert);
    
    // 5秒后自动移除
    setTimeout(() => {
        if (alert.parentElement) {
            alert.remove();
        }
    }, 5000);
}

// 显示成功消息
function showSuccessMessage(message) {
    const alert = document.createElement('div');
    alert.className = 'fixed top-4 right-4 bg-green-500 text-white px-4 py-2 rounded shadow-lg z-50';
    alert.innerHTML = `
        <div class="flex items-center">
            <i class="fas fa-check-circle mr-2"></i>
            <span>${message}</span>
            <button onclick="this.parentElement.parentElement.remove()" class="ml-4 text-white hover:text-gray-200">
                <i class="fas fa-times"></i>
            </button>
        </div>
    `;
    document.body.appendChild(alert);
    
    setTimeout(() => {
        if (alert.parentElement) {
            alert.remove();
        }
    }, 3000);
}

// 工具函数
function getStatusClass(status) {
    switch (status) {
        case 'running': return 'bg-yellow-100 text-yellow-800';
        case 'success': return 'bg-green-100 text-green-800';
        case 'failed': return 'bg-red-100 text-red-800';
        case 'published': return 'bg-blue-100 text-blue-800';
        case 'pending': return 'bg-gray-100 text-gray-800';
        case 'exhausted': return 'bg-red-100 text-red-800';
        default: return 'bg-gray-100 text-gray-800';
    }
}

function getStatusText(status) {
    switch (status) {
        case 'running': return '运行中';
        case 'success': return '成功';
        case 'failed': return '失败';
        case 'published': return '已发布';
        case 'pending': return '等待中';
        case 'exhausted': return '已耗尽';
        case 'draft': return '草稿';
        default: return status;
    }
}

function formatDateTime(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString('zh-CN');
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

// 模态框操作
function showCreateWorkflowModal() {
    document.getElementById('create-workflow-modal').classList.remove('hidden');
}

function closeCreateWorkflowModal() {
    document.getElementById('create-workflow-modal').classList.add('hidden');
    document.getElementById('create-workflow-form').reset();
}

function showExecuteModal() {
    updateWorkflowSelector();
    document.getElementById('execute-modal').classList.remove('hidden');
}

function closeExecuteModal() {
    document.getElementById('execute-modal').classList.add('hidden');
    document.getElementById('execute-workflow-form').reset();
}

function closeDetailsModal() {
    document.getElementById('details-modal').classList.add('hidden');
}

// 业务操作
async function createWorkflowTemplate() {
    const name = document.getElementById('workflow-name').value;
    const description = document.getElementById('workflow-description').value;
    const version = document.getElementById('workflow-version').value;
    
    if (!name) {
        showErrorMessage('请输入工作流名称');
        return;
    }
    
    const template = {
        name: name,
        description: description,
        version: version || '1.0.0',
        tasks: [],
        dependencies: []
    };
    
    try {
        const response = await fetch(`${API_BASE}/workflows`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(template)
        });
        
        if (response.ok) {
            showSuccessMessage('工作流模板创建成功，请使用SDK注册完整的工作流');
            closeCreateWorkflowModal();
        } else {
            const error = await response.json();
            showErrorMessage(error.error || '创建失败');
        }
    } catch (error) {
        showErrorMessage('创建工作流模板失败');
    }
}

async function executeWorkflow() {
    const workflowId = document.getElementById('execute-workflow-id').value;
    const inputText = document.getElementById('execute-input').value;
    const isAsync = document.getElementById('execute-async').checked;
    
    if (!workflowId) {
        showErrorMessage('请选择工作流');
        return;
    }
    
    let input = {};
    try {
        input = JSON.parse(inputText);
    } catch (error) {
        showErrorMessage('输入数据格式错误，请使用有效的JSON格式');
        return;
    }
    
    const request = {
        workflow_id: workflowId,
        input: input,
        async: isAsync
    };
    
    try {
        const response = await fetch(`${API_BASE}/executions`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(request)
        });
        
        if (response.ok) {
            const result = await response.json();
            if (isAsync) {
                showSuccessMessage(`异步执行已启动: ${result.execution_id}`);
            } else {
                showSuccessMessage('工作流执行完成');
            }
            closeExecuteModal();
            loadExecutions();
        } else {
            const error = await response.json();
            showErrorMessage(error.error || '执行失败');
        }
    } catch (error) {
        showErrorMessage('执行工作流失败');
    }
}

function executeWorkflowById(workflowId) {
    document.getElementById('execute-workflow-id').value = workflowId;
    showExecuteModal();
}

async function deleteWorkflow(workflowId) {
    if (!confirm('确定要删除这个工作流吗？')) {
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE}/workflows/${workflowId}`, {
            method: 'DELETE'
        });
        
        if (response.ok) {
            showSuccessMessage('工作流删除成功');
            loadWorkflows();
        } else {
            const error = await response.json();
            showErrorMessage(error.error || '删除失败');
        }
    } catch (error) {
        showErrorMessage('删除工作流失败');
    }
}

async function cancelExecution(executionId) {
    if (!confirm('确定要取消这个执行吗？')) {
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE}/executions/${executionId}/cancel`, {
            method: 'POST'
        });
        
        if (response.ok) {
            showSuccessMessage('执行已取消');
            loadExecutions();
        } else {
            const error = await response.json();
            showErrorMessage(error.error || '取消失败');
        }
    } catch (error) {
        showErrorMessage('取消执行失败');
    }
}

async function manualRetry(executionId) {
    if (!confirm('确定要手动重试这个执行吗？')) {
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE}/retries/${executionId}/retry`, {
            method: 'POST'
        });
        
        if (response.ok) {
            showSuccessMessage('重试已启动');
            loadRetries();
            loadExecutions();
        } else {
            const error = await response.json();
            showErrorMessage(error.error || '重试失败');
        }
    } catch (error) {
        showErrorMessage('手动重试失败');
    }
}

function refreshRetries() {
    loadRetries();
}

// 详情查看
async function viewWorkflowDetails(workflowId) {
    try {
        const response = await fetch(`${API_BASE}/workflows/${workflowId}`);
        const workflow = await response.json();
        
        const content = `
            <div class="space-y-4">
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label class="block text-sm font-medium text-gray-700">ID</label>
                        <div class="text-sm text-gray-900">${workflow.id}</div>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">名称</label>
                        <div class="text-sm text-gray-900">${workflow.name}</div>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">版本</label>
                        <div class="text-sm text-gray-900">${workflow.version}</div>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">状态</label>
                        <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusClass(workflow.status)}">
                            ${getStatusText(workflow.status)}
                        </span>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">任务数量</label>
                        <div class="text-sm text-gray-900">${workflow.task_count}</div>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">创建时间</label>
                        <div class="text-sm text-gray-900">${formatDateTime(workflow.created_at)}</div>
                    </div>
                </div>
                <div>
                    <label class="block text-sm font-medium text-gray-700">描述</label>
                    <div class="text-sm text-gray-900 mt-1">${workflow.description || '无描述'}</div>
                </div>
            </div>
        `;
        
        document.getElementById('details-title').textContent = `工作流详情 - ${workflow.name}`;
        document.getElementById('details-content').innerHTML = content;
        document.getElementById('details-modal').classList.remove('hidden');
        
    } catch (error) {
        showErrorMessage('获取工作流详情失败');
    }
}

async function viewExecutionDetails(executionId) {
    try {
        const response = await fetch(`${API_BASE}/executions/${executionId}`);
        const execution = await response.json();
        
        const content = `
            <div class="space-y-4">
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label class="block text-sm font-medium text-gray-700">执行ID</label>
                        <div class="text-sm text-gray-900">${execution.id}</div>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">工作流ID</label>
                        <div class="text-sm text-gray-900">${execution.workflow_id}</div>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">状态</label>
                        <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusClass(execution.status)}">
                            ${getStatusText(execution.status)}
                        </span>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">重试次数</label>
                        <div class="text-sm text-gray-900">${execution.retry_count}</div>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">开始时间</label>
                        <div class="text-sm text-gray-900">${formatDateTime(execution.started_at)}</div>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">结束时间</label>
                        <div class="text-sm text-gray-900">${execution.ended_at ? formatDateTime(execution.ended_at) : '未结束'}</div>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">耗时</label>
                        <div class="text-sm text-gray-900">${execution.duration ? formatDuration(execution.duration) : '-'}</div>
                    </div>
                </div>
                <div>
                    <label class="block text-sm font-medium text-gray-700">输入数据</label>
                    <div class="text-sm text-gray-900 mt-1 bg-gray-100 p-3 rounded">
                        <pre>${JSON.stringify(execution.input, null, 2)}</pre>
                    </div>
                </div>
                ${execution.output ? `
                <div>
                    <label class="block text-sm font-medium text-gray-700">输出数据</label>
                    <div class="text-sm text-gray-900 mt-1 bg-gray-100 p-3 rounded">
                        <pre>${JSON.stringify(execution.output, null, 2)}</pre>
                    </div>
                </div>
                ` : ''}
            </div>
        `;
        
        document.getElementById('details-title').textContent = `执行详情 - ${executionId.substring(0, 8)}...`;
        document.getElementById('details-content').innerHTML = content;
        document.getElementById('details-modal').classList.remove('hidden');
        
    } catch (error) {
        showErrorMessage('获取执行详情失败');
    }
}

async function viewExecutionLogs(executionId) {
    try {
        const response = await fetch(`${API_BASE}/executions/${executionId}/logs`);
        const logs = await response.json();
        
        let content = '<div class="space-y-2">';
        
        if (!logs || logs.length === 0) {
            content += '<div class="text-center text-gray-500 py-4">暂无日志</div>';
        } else {
            logs.forEach(log => {
                content += `
                    <div class="border border-gray-200 rounded p-3">
                        <div class="flex justify-between items-start mb-2">
                            <div class="flex items-center space-x-2">
                                <span class="text-xs font-medium text-gray-500">${formatDateTime(log.timestamp)}</span>
                                <span class="px-2 py-1 text-xs rounded ${getLogLevelClass(log.level)}">${log.level.toUpperCase()}</span>
                                ${log.task_id ? `<span class="text-xs text-gray-500">Task: ${log.task_id}</span>` : ''}
                            </div>
                        </div>
                        <div class="text-sm text-gray-900">${log.message}</div>
                    </div>
                `;
            });
        }
        
        content += '</div>';
        
        document.getElementById('details-title').textContent = `执行日志 - ${executionId.substring(0, 8)}...`;
        document.getElementById('details-content').innerHTML = content;
        document.getElementById('details-modal').classList.remove('hidden');
        
    } catch (error) {
        showErrorMessage('获取执行日志失败');
    }
}

async function viewRetryDetails(executionId) {
    try {
        const response = await fetch(`${API_BASE}/retries/${executionId}`);
        const retry = await response.json();
        
        const content = `
            <div class="space-y-4">
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label class="block text-sm font-medium text-gray-700">执行ID</label>
                        <div class="text-sm text-gray-900">${retry.execution_id}</div>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">工作流ID</label>
                        <div class="text-sm text-gray-900">${retry.workflow_id}</div>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">重试次数</label>
                        <div class="text-sm text-gray-900">${retry.retry_count}</div>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">状态</label>
                        <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusClass(retry.status)}">
                            ${getStatusText(retry.status)}
                        </span>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">最后重试时间</label>
                        <div class="text-sm text-gray-900">${formatDateTime(retry.last_retry_at)}</div>
                    </div>
                </div>
                <div>
                    <label class="block text-sm font-medium text-gray-700">失败原因</label>
                    <div class="text-sm text-gray-900 mt-1 bg-red-50 p-3 rounded border border-red-200">
                        ${retry.failure_reason}
                    </div>
                </div>
            </div>
        `;
        
        document.getElementById('details-title').textContent = `重试详情 - ${executionId.substring(0, 8)}...`;
        document.getElementById('details-content').innerHTML = content;
        document.getElementById('details-modal').classList.remove('hidden');
        
    } catch (error) {
        showErrorMessage('获取重试详情失败');
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
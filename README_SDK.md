# Task Graph SDK 使用指南

Task Graph SDK 是一个强大且易用的Go任务编排引擎SDK，提供了简化的API接口，让开发者能够轻松地集成和使用任务编排功能。

## 🚀 快速开始

### 安装

```bash
go get github.com/XXueTu/graph_task
```

### 一行代码启动

```go
package main

import (
    "log"
    "github.com/XXueTu/graph_task/interfaces/sdk"
)

func main() {
    // 一行代码启动带Web界面的任务引擎
    client, err := sdk.QuickStart("user:pass@tcp(localhost:3306)/taskdb", 8080)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // 服务已启动，Web管理界面: http://localhost:8080
    select {} // 保持运行
}
```

### 最简单的任务执行

```go
// 定义任务处理函数
func myTask(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
    name := input["name"].(string)
    return map[string]interface{}{
        "greeting": "Hello, " + name + "!",
    }, nil
}

// 一行代码执行任务
result, err := sdk.OneShot("user:pass@tcp(localhost:3306)/taskdb", myTask, map[string]interface{}{
    "name": "World",
})
```

## 📚 基本概念

### Client（客户端）
SDK的核心组件，提供所有功能的入口点。

### Workflow（工作流）
由多个任务组成的有向无环图，定义了任务的执行顺序和依赖关系。

### Task（任务）
工作流中的基本执行单元，包含具体的业务逻辑。

### Execution（执行）
工作流的一次运行实例，包含输入数据、输出结果和执行状态。

## 🛠️ 详细使用指南

### 1. 创建客户端

#### 方式一：快速启动（推荐）
```go
client, err := sdk.QuickStart("your-database-dsn", 8080)
```

#### 方式二：使用默认配置
```go
client, err := sdk.NewSimpleClient()
```

#### 方式三：使用自定义配置
```go
config := &sdk.ClientConfig{
    MySQLDSN:          "user:pass@tcp(localhost:3306)/taskdb",
    WebPort:           8080,
    EnableWeb:         true,
    MaxConcurrency:    20,
    MaxAutoRetries:    5,
    RetryDelay:        2 * time.Second,
    BackoffMultiplier: 2.0,
    MaxRetryDelay:     1 * time.Minute,
}

client, err := sdk.NewClient(config)
```

### 2. 创建工作流

#### 简化方式（推荐）
```go
workflow := client.NewEasyWorkflow("data-processing").
    Task("extract", "数据提取", extractHandler).
    Task("transform", "数据转换", transformHandler).
    Task("load", "数据加载", loadHandler).
    Sequential("extract", "transform", "load") // 顺序执行

// 注册工作流
err := workflow.Register()
```

#### 并行任务
```go
workflow := client.NewEasyWorkflow("parallel-processing").
    Task("task1", "任务1", handler1).
    Task("task2", "任务2", handler2).
    Task("task3", "任务3", handler3).
    Parallel("task1", "task2", "task3") // 并行执行
```

#### 复杂依赖关系
```go
workflow := client.NewEasyWorkflow("complex-workflow").
    Task("A", "任务A", handlerA).
    Task("B", "任务B", handlerB).
    Task("C", "任务C", handlerC).
    Task("D", "任务D", handlerD).
    Then("A", "B").      // A -> B
    Then("A", "C").      // A -> C  
    Then("B", "D").      // B -> D
    Then("C", "D")       // C -> D
```

### 3. 执行工作流

#### 同步执行
```go
result, err := client.Execute(ctx, "workflow-name", map[string]interface{}{
    "input": "data",
})

if err != nil {
    log.Printf("执行失败: %v", err)
} else {
    fmt.Printf("执行结果: %+v\n", result.Output)
}
```

#### 异步执行
```go
executionID, err := client.ExecuteAsync(ctx, "workflow-name", input)
if err != nil {
    log.Fatal(err)
}

// 等待完成
result, err := client.WaitForExecution(executionID, 5*time.Minute)
```

#### 注册并立即执行
```go
result, err := workflow.RegisterAndRun(input)
```

### 4. 使用预定义模板

```go
// 数据处理模板
template := sdk.DataProcessingTemplate()

handlers := map[string]workflow.TaskHandler{
    "extract":   extractHandler,
    "validate":  validateHandler,
    "transform": transformHandler,
    "load":      loadHandler,
}

workflow, err := client.CreateFromTemplate(template, handlers)
if err != nil {
    log.Fatal(err)
}

err = workflow.Register()
```

### 5. 批量执行

```go
batch := client.NewBatchExecution("my-workflow").
    SetConcurrency(10). // 设置并发数
    AddInput(input1).
    AddInput(input2).
    AddInput(input3)

results, err := batch.Execute(ctx)
```

### 6. 定时任务

```go
scheduled := client.NewScheduledExecution("daily-report", input, 24*time.Hour)

// 在goroutine中启动
go func() {
    err := scheduled.Start(ctx)
    if err != nil {
        log.Printf("定时任务失败: %v", err)
    }
}()

// 停止定时任务
scheduled.Stop()
```

### 7. 监控和日志

#### 获取执行状态
```go
status, err := client.GetExecutionStatus(executionID)
if err == nil {
    fmt.Printf("进度: %.1f%%, 状态: %s\n", status.Progress, status.Status)
}
```

#### 获取执行日志
```go
logs, err := client.GetExecutionLogs(executionID, 100, 0)
for _, log := range logs {
    fmt.Printf("[%s] %s: %s\n", log.Level, log.Timestamp, log.Message)
}
```

#### 系统监控
```go
monitor := client.NewMonitor()
stats, err := monitor.GetSystemStats()
if err == nil {
    fmt.Printf("工作流数量: %d, 执行总数: %d\n", stats.TotalWorkflows, stats.TotalExecutions)
}
```

### 8. 重试管理

#### 获取失败的执行
```go
failedExecutions, err := client.GetFailedExecutions()
for _, failed := range failedExecutions {
    fmt.Printf("失败执行: %s, 原因: %s\n", failed.ExecutionID, failed.FailureReason)
}
```

#### 手动重试
```go
retryResult, err := client.ManualRetry(ctx, executionID)
if err == nil {
    fmt.Printf("重试结果: %s\n", retryResult.Status)
}
```

## 🎯 高级功能

### 1. 条件执行

```go
condition := func(input map[string]interface{}) bool {
    return input["type"].(string) == "urgent"
}

conditional := client.NewConditionalExecution(
    condition,
    "urgent-workflow",    // 条件为真时执行
    "normal-workflow",    // 条件为假时执行
    input,
)

result, err := conditional.Execute(ctx)
```

### 2. 工作流监控

```go
monitor := client.NewMonitor()

// 实时监控执行进度
err := monitor.WatchExecution(executionID, func(status *sdk.ExecutionStatus) {
    fmt.Printf("当前进度: %.1f%%\n", status.Progress)
})
```

### 3. 环境配置

```go
var config *sdk.ClientConfig

switch os.Getenv("ENV") {
case "production":
    config = sdk.ConfigurationHelper{}.ProductionConfig(dsn)
case "development":
    config = sdk.ConfigurationHelper{}.DevelopmentConfig(dsn)
case "test":
    config = sdk.ConfigurationHelper{}.TestConfig(dsn)
default:
    config = sdk.DefaultClientConfig()
}

client, err := sdk.NewClient(config)
```

## 📝 任务处理函数编写指南

### 基本结构

```go
func myTaskHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
    // 1. 从输入中获取数据
    data := input["data"].(string)
    
    // 2. 获取上一个任务的输出（如果有依赖）
    if prevOutput, exists := input["previous_task_output"]; exists {
        prevData := prevOutput.(map[string]interface{})
        // 使用前一个任务的输出
    }
    
    // 3. 执行业务逻辑
    result := processData(data)
    
    // 4. 返回输出数据
    return map[string]interface{}{
        "result": result,
        "processed_at": time.Now(),
    }, nil
}
```

### 错误处理

```go
func riskyTaskHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
    // 可重试的错误（网络错误、临时失败等）
    if someTemporaryCondition {
        return nil, fmt.Errorf("临时错误，可以重试")
    }
    
    // 不可重试的错误（参数错误、逻辑错误等）
    if invalidInput {
        return nil, fmt.Errorf("输入参数无效，不能重试")
    }
    
    // 正常处理
    return map[string]interface{}{
        "status": "success",
    }, nil
}
```

### 长时间运行的任务

```go
func longRunningTaskHandler(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
    // 检查上下文取消
    select {
    case <-ctx.Done():
        return nil, fmt.Errorf("任务被取消")
    default:
    }
    
    // 分阶段处理，定期检查取消信号
    for i := 0; i < 100; i++ {
        select {
        case <-ctx.Done():
            return nil, fmt.Errorf("任务被取消")
        default:
            // 处理一个单元
            time.Sleep(100 * time.Millisecond)
        }
    }
    
    return map[string]interface{}{
        "completed": true,
    }, nil
}
```

## 🌐 Web管理界面

当启用Web界面时（默认启用），你可以通过浏览器访问：`http://localhost:8080`

### 功能特性：
- **工作流管理**: 查看、创建、删除工作流
- **执行监控**: 实时查看执行状态、进度、日志
- **重试管理**: 查看失败执行、手动重试
- **系统监控**: 查看系统性能指标
- **SDK集成指南**: 内置的集成文档和示例代码

## 🔧 最佳实践

### 1. 错误处理
- 区分可重试和不可重试的错误
- 使用有意义的错误消息
- 记录详细的错误日志

### 2. 性能优化
- 合理设置并发数
- 避免任务间的紧耦合
- 使用批量执行处理大量数据

### 3. 监控和运维
- 定期检查失败的执行
- 监控系统资源使用情况
- 设置合理的重试策略

### 4. 测试
- 为每个任务编写单元测试
- 使用测试配置进行集成测试
- 验证工作流的完整性

## 📋 完整示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/XXueTu/graph_task/interfaces/sdk"
    "github.com/XXueTu/graph_task/domain/workflow"
)

func main() {
    // 1. 启动服务
    client, err := sdk.QuickStart("user:pass@tcp(localhost:3306)/taskdb", 8080)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // 2. 创建工作流
    wf := client.NewEasyWorkflow("user-onboarding").
        Task("validate", "验证用户信息", validateUser).
        Task("create", "创建用户", createUser).
        Task("notify", "发送通知", sendNotification).
        Sequential("validate", "create", "notify")
    
    // 3. 注册并执行
    result, err := wf.RegisterAndRun(map[string]interface{}{
        "email": "user@example.com",
        "name":  "张三",
    })
    
    if err != nil {
        log.Printf("执行失败: %v", err)
    } else {
        fmt.Printf("用户注册成功: %+v\n", result.Output)
    }
    
    // 4. Web界面已可用：http://localhost:8080
    fmt.Println("🌐 访问 http://localhost:8080 查看管理界面")
    
    // 保持服务运行
    select {}
}

// 任务处理函数
func validateUser(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
    email := input["email"].(string)
    name := input["name"].(string)
    
    if email == "" || name == "" {
        return nil, fmt.Errorf("邮箱和姓名不能为空")
    }
    
    return map[string]interface{}{
        "email":      email,
        "name":       name,
        "validated":  true,
        "created_at": time.Now(),
    }, nil
}

func createUser(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
    validateOutput := input["validate_output"].(map[string]interface{})
    
    userID := fmt.Sprintf("user_%d", time.Now().Unix())
    
    return map[string]interface{}{
        "user_id": userID,
        "email":   validateOutput["email"],
        "name":    validateOutput["name"],
        "status":  "active",
    }, nil
}

func sendNotification(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
    createOutput := input["create_output"].(map[string]interface{})
    
    // 模拟发送通知
    time.Sleep(100 * time.Millisecond)
    
    return map[string]interface{}{
        "notification_sent": true,
        "user_id":          createOutput["user_id"],
        "sent_at":          time.Now(),
    }, nil
}
```

## 🤝 技术支持

- **GitHub**: https://github.com/XXueTu/graph_task
- **文档**: 查看 CLAUDE.md 和本文档
- **示例代码**: interfaces/sdk/examples.go
- **Web界面**: 启动服务后访问管理界面

## 📄 许可证

本项目采用 MIT 许可证。详情请查看 LICENSE 文件。
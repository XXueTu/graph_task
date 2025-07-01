# Task Graph 集成指南

本指南详细说明如何将 Task Graph 任务编排引擎集成到现有系统中。

## 🏗️ 集成架构

### 1. 包依赖集成（推荐）

Task Graph 设计为Go包，可以直接集成到现有应用中：

```go
// 在你的主应用中
package main

import (
    "github.com/XXueTu/graph_task/interfaces/sdk"
    // 你的其他导入
)

func main() {
    // 初始化你的应用
    app := initializeApp()
    
    // 集成Task Graph
    taskEngine, err := sdk.QuickStart(app.DatabaseDSN, 8080)
    if err != nil {
        log.Fatal(err)
    }
    defer taskEngine.Close()
    
    // 注册业务工作流
    registerBusinessWorkflows(taskEngine)
    
    // 启动你的应用
    app.Start()
}
```

### 2. 微服务集成

作为独立的微服务运行：

```bash
# 启动Task Graph服务
./task_graph_server -dsn="user:pass@tcp(localhost:3306)/taskdb" -port=8080

# 通过HTTP API集成
curl -X POST http://localhost:8080/api/v1/executions \
  -H "Content-Type: application/json" \
  -d '{"workflow_id": "data-processing", "input": {"data": "test"}}'
```

### 3. SDK客户端集成

使用SDK连接到远程Task Graph服务：

```go
// 连接到远程Task Graph服务
client := sdk.NewRemoteClient("http://task-graph-service:8080")
result, err := client.Execute(ctx, "workflow-id", input)
```

## 🔌 集成模式

### 1. 嵌入式集成

将Task Graph完全嵌入到现有应用中：

```go
type MyApplication struct {
    db         *sql.DB
    taskEngine *sdk.Client
    // 其他组件
}

func NewApplication(dsn string) (*MyApplication, error) {
    // 初始化数据库
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }
    
    // 初始化Task Graph（使用相同的数据库）
    taskEngine, err := sdk.NewClientWithDSN(dsn)
    if err != nil {
        return nil, err
    }
    
    app := &MyApplication{
        db:         db,
        taskEngine: taskEngine,
    }
    
    // 注册业务工作流
    app.registerWorkflows()
    
    return app, nil
}

func (app *MyApplication) registerWorkflows() {
    // 用户注册流程
    app.taskEngine.NewEasyWorkflow("user-registration").
        Task("validate", "验证", app.validateUser).
        Task("create", "创建", app.createUser).
        Task("notify", "通知", app.notifyUser).
        Sequential("validate", "create", "notify").
        Register()
    
    // 订单处理流程
    app.taskEngine.NewEasyWorkflow("order-processing").
        Task("validate_order", "验证订单", app.validateOrder).
        Task("process_payment", "处理支付", app.processPayment).
        Task("update_inventory", "更新库存", app.updateInventory).
        Task("send_confirmation", "发送确认", app.sendConfirmation).
        Sequential("validate_order", "process_payment", "update_inventory", "send_confirmation").
        Register()
}

// 业务方法，同时也是任务处理器
func (app *MyApplication) validateUser(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
    // 使用应用的数据库连接
    email := input["email"].(string)
    
    var count int
    err := app.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&count)
    if err != nil {
        return nil, err
    }
    
    if count > 0 {
        return nil, fmt.Errorf("用户已存在")
    }
    
    return map[string]interface{}{
        "email": email,
        "valid": true,
    }, nil
}
```

### 2. 事件驱动集成

通过事件系统集成：

```go
type EventDrivenApp struct {
    eventBus   *EventBus
    taskEngine *sdk.Client
}

func (app *EventDrivenApp) setupIntegration() {
    // 监听业务事件，触发工作流
    app.eventBus.Subscribe("user.registered", func(event Event) {
        app.taskEngine.ExecuteAsync(context.Background(), "user-onboarding", map[string]interface{}{
            "user_id": event.Data["user_id"],
            "email":   event.Data["email"],
        })
    })
    
    app.eventBus.Subscribe("order.created", func(event Event) {
        app.taskEngine.ExecuteAsync(context.Background(), "order-processing", map[string]interface{}{
            "order_id": event.Data["order_id"],
            "amount":   event.Data["amount"],
        })
    })
    
    // 监听工作流完成事件
    app.taskEngine.Subscribe("workflow.completed", func(event *application.Event) {
        // 处理工作流完成后的逻辑
        if event.WorkflowID() == "order-processing" {
            app.eventBus.Publish(Event{
                Type: "order.processed",
                Data: event.Data(),
            })
        }
    })
}
```

### 3. HTTP API集成

通过HTTP API与外部系统集成：

```go
type APIIntegration struct {
    client     *http.Client
    taskEngine *sdk.Client
}

func (api *APIIntegration) setupWebhookHandlers() {
    http.HandleFunc("/webhooks/payment", api.handlePaymentWebhook)
    http.HandleFunc("/webhooks/shipment", api.handleShipmentWebhook)
}

func (api *APIIntegration) handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
    var webhook PaymentWebhook
    json.NewDecoder(r.Body).Decode(&webhook)
    
    // 触发支付处理工作流
    api.taskEngine.ExecuteAsync(context.Background(), "payment-processing", map[string]interface{}{
        "payment_id": webhook.PaymentID,
        "amount":     webhook.Amount,
        "status":     webhook.Status,
    })
    
    w.WriteHeader(http.StatusOK)
}
```

## 📦 部署配置

### 1. Docker集成

```dockerfile
# Dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o myapp .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/myapp .
COPY --from=builder /app/interfaces/web/static ./web/static

EXPOSE 8080
CMD ["./myapp"]
```

```yaml
# docker-compose.yml
version: '3.8'
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: password
      MYSQL_DATABASE: taskdb
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql

  myapp:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_DSN=root:password@tcp(mysql:3306)/taskdb?charset=utf8mb4&parseTime=True&loc=Local
    depends_on:
      - mysql
    volumes:
      - ./web/static:/root/web/static

volumes:
  mysql_data:
```

### 2. Kubernetes部署

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp-with-taskgraph
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: myapp
        image: myapp:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_DSN
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: dsn
        - name: WEB_PORT
          value: "8080"
---
apiVersion: v1
kind: Service
metadata:
  name: myapp-service
spec:
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

### 3. 配置管理

```go
type Config struct {
    // 应用配置
    AppPort    int    `env:"APP_PORT" default:"3000"`
    AppName    string `env:"APP_NAME" default:"MyApp"`
    
    // Task Graph配置
    TaskGraphDSN     string `env:"TASKGRAPH_DSN" required:"true"`
    TaskGraphWebPort int    `env:"TASKGRAPH_WEB_PORT" default:"8080"`
    TaskGraphEnabled bool   `env:"TASKGRAPH_ENABLED" default:"true"`
    
    // 其他配置...
}

func (c *Config) CreateTaskGraphClient() (*sdk.Client, error) {
    if !c.TaskGraphEnabled {
        return nil, nil // 禁用Task Graph
    }
    
    config := &sdk.ClientConfig{
        MySQLDSN:       c.TaskGraphDSN,
        WebPort:        c.TaskGraphWebPort,
        EnableWeb:      true,
        MaxConcurrency: 10,
    }
    
    return sdk.NewClient(config)
}
```

## 🔄 数据流集成

### 1. 数据库共享

```go
// 共享数据库连接
type SharedDatabase struct {
    db *sql.DB
}

func (sdb *SharedDatabase) CreateTaskHandler() workflow.TaskHandler {
    return func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
        // 使用共享的数据库连接
        userID := input["user_id"].(int)
        
        var user User
        err := sdb.db.QueryRow("SELECT id, name, email FROM users WHERE id = ?", userID).
            Scan(&user.ID, &user.Name, &user.Email)
        if err != nil {
            return nil, err
        }
        
        return map[string]interface{}{
            "user": user,
        }, nil
    }
}
```

### 2. 消息队列集成

```go
// Redis队列集成
type RedisQueueIntegration struct {
    redis      *redis.Client
    taskEngine *sdk.Client
}

func (rqi *RedisQueueIntegration) processQueue(queueName string) {
    go func() {
        for {
            // 从队列中获取任务
            result, err := rqi.redis.BLPop(context.Background(), 0, queueName).Result()
            if err != nil {
                continue
            }
            
            var task QueueTask
            json.Unmarshal([]byte(result[1]), &task)
            
            // 提交到Task Graph执行
            rqi.taskEngine.ExecuteAsync(context.Background(), task.WorkflowID, task.Input)
        }
    }()
}
```

### 3. 缓存集成

```go
type CacheIntegration struct {
    cache      *redis.Client
    taskEngine *sdk.Client
}

func (ci *CacheIntegration) createCacheAwareTask() workflow.TaskHandler {
    return func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
        key := fmt.Sprintf("task_result:%s", input["key"])
        
        // 检查缓存
        cached, err := ci.cache.Get(context.Background(), key).Result()
        if err == nil {
            var result map[string]interface{}
            json.Unmarshal([]byte(cached), &result)
            return result, nil
        }
        
        // 执行实际业务逻辑
        result := map[string]interface{}{
            "processed": true,
            "data":      processData(input),
        }
        
        // 缓存结果
        resultJson, _ := json.Marshal(result)
        ci.cache.Set(context.Background(), key, resultJson, time.Hour)
        
        return result, nil
    }
}
```

## 🔒 安全集成

### 1. 认证集成

```go
type AuthIntegration struct {
    jwtSecret  []byte
    taskEngine *sdk.Client
}

func (ai *AuthIntegration) createAuthenticatedTask() workflow.TaskHandler {
    return func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
        // 验证JWT token
        tokenString := input["auth_token"].(string)
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return ai.jwtSecret, nil
        })
        
        if err != nil || !token.Valid {
            return nil, fmt.Errorf("认证失败")
        }
        
        claims := token.Claims.(jwt.MapClaims)
        userID := claims["user_id"].(string)
        
        // 执行需要认证的业务逻辑
        return map[string]interface{}{
            "user_id": userID,
            "result":  "authenticated operation completed",
        }, nil
    }
}
```

### 2. 权限控制

```go
func (ai *AuthIntegration) createAuthorizedTask(requiredPermission string) workflow.TaskHandler {
    return func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
        userID := input["user_id"].(string)
        
        // 检查权限
        hasPermission, err := ai.checkPermission(userID, requiredPermission)
        if err != nil {
            return nil, err
        }
        
        if !hasPermission {
            return nil, fmt.Errorf("权限不足")
        }
        
        // 执行需要权限的操作
        return ai.executePrivilegedOperation(input)
    }
}
```

## 📊 监控集成

### 1. Prometheus监控

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    workflowExecutions = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "taskgraph_workflow_executions_total",
            Help: "Total number of workflow executions",
        },
        []string{"workflow_id", "status"},
    )
    
    executionDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "taskgraph_execution_duration_seconds",
            Help: "Duration of workflow executions",
        },
        []string{"workflow_id"},
    )
)

func (app *MyApp) setupMonitoring() {
    // 监听工作流事件
    app.taskEngine.Subscribe("workflow.completed", func(event *application.Event) {
        workflowExecutions.WithLabelValues(event.WorkflowID(), "completed").Inc()
        
        if duration, ok := event.Data()["duration"].(time.Duration); ok {
            executionDuration.WithLabelValues(event.WorkflowID()).Observe(duration.Seconds())
        }
    })
    
    app.taskEngine.Subscribe("workflow.failed", func(event *application.Event) {
        workflowExecutions.WithLabelValues(event.WorkflowID(), "failed").Inc()
    })
}
```

### 2. 日志集成

```go
import "github.com/sirupsen/logrus"

type LogIntegration struct {
    logger     *logrus.Logger
    taskEngine *sdk.Client
}

func (li *LogIntegration) setupLogging() {
    // 设置结构化日志
    li.logger.SetFormatter(&logrus.JSONFormatter{})
    
    // 创建带日志的任务处理器
    li.taskEngine.NewEasyWorkflow("logged-workflow").
        Task("process", "处理", li.createLoggedTask("process")).
        Register()
}

func (li *LogIntegration) createLoggedTask(taskName string) workflow.TaskHandler {
    return func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
        li.logger.WithFields(logrus.Fields{
            "task":         taskName,
            "execution_id": ctx.ExecutionID(),
            "input":        input,
        }).Info("任务开始执行")
        
        start := time.Now()
        result, err := li.executeBusinessLogic(input)
        duration := time.Since(start)
        
        if err != nil {
            li.logger.WithFields(logrus.Fields{
                "task":         taskName,
                "execution_id": ctx.ExecutionID(),
                "error":        err.Error(),
                "duration":     duration,
            }).Error("任务执行失败")
            return nil, err
        }
        
        li.logger.WithFields(logrus.Fields{
            "task":         taskName,
            "execution_id": ctx.ExecutionID(),
            "duration":     duration,
            "output":       result,
        }).Info("任务执行成功")
        
        return result, nil
    }
}
```

## 🧪 测试集成

### 1. 单元测试

```go
func TestWorkflowIntegration(t *testing.T) {
    // 创建测试客户端
    config := sdk.ConfigurationHelper{}.TestConfig(":memory:")
    client, err := sdk.NewClient(config)
    require.NoError(t, err)
    defer client.Close()
    
    // 注册测试工作流
    workflow := client.NewEasyWorkflow("test-workflow").
        Task("task1", "测试任务1", func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
            return map[string]interface{}{
                "result": "test",
            }, nil
        })
    
    err = workflow.Register()
    require.NoError(t, err)
    
    // 执行测试
    result, err := client.SimpleExecute("test-workflow", map[string]interface{}{
        "input": "test",
    })
    
    require.NoError(t, err)
    assert.Equal(t, "success", result.Status)
    assert.Equal(t, "test", result.Output["result"])
}
```

### 2. 集成测试

```go
func TestFullIntegration(t *testing.T) {
    // 启动测试数据库
    db := setupTestDatabase(t)
    defer db.Close()
    
    // 创建应用实例
    app, err := NewApplication(db.DSN())
    require.NoError(t, err)
    defer app.Close()
    
    // 测试完整的业务流程
    result, err := app.ProcessUserRegistration(map[string]interface{}{
        "email": "test@example.com",
        "name":  "Test User",
    })
    
    require.NoError(t, err)
    assert.True(t, result.IsSuccess())
    
    // 验证数据库状态
    var count int
    db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "test@example.com").Scan(&count)
    assert.Equal(t, 1, count)
}
```

## 📋 迁移指南

### 从现有任务系统迁移

1. **评估现有系统**
   - 识别现有的任务/工作流
   - 分析依赖关系
   - 评估数据流

2. **分步迁移**
   ```go
   // 第一步：并行运行
   func (app *App) processOrder(orderID string) {
       // 保留原有逻辑
       app.legacyOrderProcessor.Process(orderID)
       
       // 同时使用新系统（仅记录，不执行关键操作）
       app.taskEngine.ExecuteAsync(context.Background(), "order-processing-test", map[string]interface{}{
           "order_id": orderID,
           "mode":     "test",
       })
   }
   
   // 第二步：逐步替换
   func (app *App) processOrder(orderID string) {
       if app.config.UseNewSystem {
           result, err := app.taskEngine.Execute(context.Background(), "order-processing", map[string]interface{}{
               "order_id": orderID,
           })
           if err != nil {
               // 回退到旧系统
               app.legacyOrderProcessor.Process(orderID)
           }
       } else {
           app.legacyOrderProcessor.Process(orderID)
       }
   }
   ```

3. **数据迁移**
   ```sql
   -- 迁移现有任务状态
   INSERT INTO task_graph_executions (id, workflow_id, status, input, output, created_at)
   SELECT 
       CONCAT('migrated_', id),
       'legacy-task',
       CASE status 
           WHEN 'completed' THEN 'success'
           WHEN 'failed' THEN 'failed'
           ELSE 'pending'
       END,
       JSON_OBJECT('legacy_id', id, 'data', task_data),
       JSON_OBJECT('result', result_data),
       created_at
   FROM legacy_tasks;
   ```

## 🚀 最佳实践总结

1. **渐进式集成**: 不要一次性替换所有功能，采用渐进式的方法
2. **保持简单**: 从简单的工作流开始，逐步增加复杂性
3. **监控和日志**: 从一开始就集成监控和日志系统
4. **错误处理**: 设计好的错误处理和重试机制
5. **测试驱动**: 为每个工作流编写测试
6. **文档化**: 记录所有的工作流和集成点
7. **性能优化**: 监控性能，及时优化瓶颈

通过以上指南，你可以成功地将Task Graph集成到现有系统中，提升系统的可维护性、可扩展性和可观察性。
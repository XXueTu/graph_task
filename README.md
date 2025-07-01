# Task Graph - 高性能任务编排系统

一个基于Go的高性能任务编排、管理和执行系统，支持DSL构建、可视化管理、并发执行和多副本部署。

## 核心特性

- **DSL构建器**: 支持通过DSL定义任务流
- **高性能执行引擎**: 基于DAG的并发任务执行
- **可视化管理**: Web界面支持任务全生命周期管理
- **SDK集成**: 提供Go SDK供客户端集成
- **多副本部署**: 支持分布式部署和负载均衡
- **高并发优化**: 针对并发场景进行性能优化
- **最小依赖**: 尽可能减少外部依赖

## 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Console   │    │   SDK Client    │    │   HTTP Client   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   API Gateway   │
                    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │  Task Graph     │
                    │   Engine        │
                    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │   Storage       │
                    │ MySQL + Cache   │
                    └─────────────────┘
```

## 快速开始

```go
package main

import (
    "github.com/XXueTu/graph_task"
)

func main() {
    // 创建引擎
    engine := graph_task.NewEngine()
    
    // 定义工作流
    workflow := engine.NewWorkflow("example").
        AddTask("task1", func() error { return nil }).
        AddTask("task2", func() error { return nil }).
        AddDependency("task1", "task2")
    
    // 执行工作流
    result := engine.Execute(workflow)
    fmt.Println("执行结果:", result)
}
```
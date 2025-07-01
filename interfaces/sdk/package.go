// Package sdk 提供Task Graph引擎的Go SDK
//
// 这个包为Task Graph任务编排引擎提供了简化的Go SDK接口，
// 让开发者能够轻松地集成和使用任务编排功能。
//
// 主要特性：
//   - 简化的客户端接口
//   - 内置Web管理界面
//   - 自动重试机制
//   - 批量执行支持
//   - 定时任务调度
//   - 工作流模板
//   - 实时监控
//
// 快速开始：
//
//	// 1. 快速启动（一行代码启动带Web界面的引擎）
//	client, err := sdk.QuickStart("user:pass@tcp(localhost:3306)/taskdb", 8080)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// 2. 创建简单工作流
//	workflow := client.NewEasyWorkflow("hello-world").
//	    Task("greet", "打招呼", greetHandler).
//	    Task("process", "处理", processHandler).
//	    Task("finish", "完成", finishHandler).
//	    Sequential("greet", "process", "finish")
//
//	// 3. 注册并执行
//	result, err := workflow.RegisterAndRun(map[string]interface{}{
//	    "name": "世界",
//	})
//
// 更多功能示例，请参考examples.go文件。
//
package sdk

const (
	// Version SDK版本
	Version = "1.0.0"
	
	// UserAgent 用户代理
	UserAgent = "Task-Graph-SDK/1.0.0"
)
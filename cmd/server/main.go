package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/XXueTu/graph_task/domain/workflow"
	"github.com/XXueTu/graph_task/interfaces/sdk"
)

func main() {
	var (
		dsn      = flag.String("dsn", "root:Root@123@tcp(10.99.51.9:3306)/graph_task?charset=utf8mb4&parseTime=True&loc=Local", "MySQL DSN")
		port     = flag.Int("port", 8088, "Web server port")
		example  = flag.String("example", "", "Run example (quick-start, advanced, retry, monitoring)")
		noDaemon = flag.Bool("no-daemon", false, "Don't run as daemon, exit after example")
	)
	flag.Parse()

	fmt.Println("🚀 Task Graph Server Starting...")

	// 如果指定了示例，运行示例
	if *example != "" {
		if err := runExample(*example, *dsn); err != nil {
			log.Fatalf("运行示例失败: %v", err)
		}

		if *noDaemon {
			return
		}
	}

	// 启动服务器
	client, err := sdk.QuickStart(*dsn, *port)
	if err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
	defer client.Close()

	// 注册示例工作流
	if err := registerExampleWorkflows(client); err != nil {
		log.Printf("注册示例工作流失败: %v", err)
	}

	fmt.Printf("✅ 服务器已启动\n")
	fmt.Printf("📊 Web管理界面: %s\n", client.GetWebURL())
	fmt.Println("📖 API文档: http://localhost:8088/api/v1/")
	fmt.Println("🔄 按 Ctrl+C 停止服务器")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n🛑 正在停止服务器...")
}

func runExample(exampleName, dsn string) error {
	switch exampleName {
	case "quick-start":
		return sdk.QuickStartExample()
	case "advanced":
		return sdk.AdvancedExample()
	case "retry":
		return sdk.RetryExample()
	case "monitoring":
		return sdk.MonitoringExample()
	default:
		return fmt.Errorf("未知示例: %s", exampleName)
	}
}

func registerExampleWorkflows(client *sdk.Client) error {
	// 注册数据处理工作流示例
	err := registerDataProcessingWorkflow(client)
	if err != nil {
		return fmt.Errorf("注册数据处理工作流失败: %w", err)
	}

	// 注册用户注册工作流示例
	err = registerUserRegistrationWorkflow(client)
	if err != nil {
		return fmt.Errorf("注册用户注册工作流失败: %w", err)
	}

	fmt.Println("✅ 示例工作流已注册")
	return nil
}

func registerDataProcessingWorkflow(client *sdk.Client) error {
	// 使用模板创建数据处理工作流
	template := sdk.DataProcessingTemplate()

	handlers := map[string]workflow.TaskHandler{
		"extract": func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
			time.Sleep(100 * time.Millisecond) // 模拟处理时间

			source := input["source"].(string)
			return map[string]interface{}{
				"records":      1000,
				"source":       source,
				"extracted_at": time.Now(),
			}, nil
		},
		"validate": func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
			time.Sleep(50 * time.Millisecond)

			extractOutput := input["extract_output"].(map[string]interface{})
			records := extractOutput["records"].(int)

			validRecords := int(float64(records) * 0.95) // 95% 有效
			return map[string]interface{}{
				"total_records":   records,
				"valid_records":   validRecords,
				"invalid_records": records - validRecords,
			}, nil
		},
		"transform": func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
			time.Sleep(200 * time.Millisecond)

			validateOutput := input["validate_output"].(map[string]interface{})
			validRecords := validateOutput["valid_records"].(int)

			return map[string]interface{}{
				"transformed_records": validRecords,
				"transformation_type": "normalize",
				"transformed_at":      time.Now(),
			}, nil
		},
		"load": func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
			time.Sleep(150 * time.Millisecond)

			transformOutput := input["transform_output"].(map[string]interface{})
			transformedRecords := transformOutput["transformed_records"].(int)

			return map[string]interface{}{
				"loaded_records": transformedRecords,
				"target_table":   "processed_data",
				"loaded_at":      time.Now(),
				"status":         "success",
			}, nil
		},
	}

	workflow, err := client.CreateFromTemplate(template, handlers)
	if err != nil {
		return err
	}

	return workflow.Register()
}

func registerUserRegistrationWorkflow(client *sdk.Client) error {
	workflow := client.NewEasyWorkflow("user-registration").
		Task("validate_email", "验证邮箱", func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
			email := input["email"].(string)
			time.Sleep(50 * time.Millisecond)

			// 简单的邮箱验证
			if !contains(email, "@") {
				return nil, fmt.Errorf("无效的邮箱地址: %s", email)
			}

			return map[string]interface{}{
				"email":        email,
				"email_valid":  true,
				"validated_at": time.Now(),
			}, nil
		}).
		Task("check_duplicate", "检查重复", func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
			validateOutput := input["validate_email_output"].(map[string]interface{})
			email := validateOutput["email"].(string)
			time.Sleep(100 * time.Millisecond)

			// 模拟数据库检查
			duplicate := false
			if email == "test@example.com" {
				duplicate = true
			}

			return map[string]interface{}{
				"email":        email,
				"is_duplicate": duplicate,
				"checked_at":   time.Now(),
			}, nil
		}).
		Task("create_user", "创建用户", func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
			checkOutput := input["check_duplicate_output"].(map[string]interface{})
			email := checkOutput["email"].(string)
			isDuplicate := checkOutput["is_duplicate"].(bool)

			if isDuplicate {
				return nil, fmt.Errorf("用户已存在: %s", email)
			}

			time.Sleep(200 * time.Millisecond)

			userID := fmt.Sprintf("user_%d", time.Now().Unix())
			return map[string]interface{}{
				"user_id":    userID,
				"email":      email,
				"created_at": time.Now(),
				"status":     "active",
			}, nil
		}).
		Task("send_welcome_email", "发送欢迎邮件", func(ctx workflow.ExecContext, input map[string]interface{}) (map[string]interface{}, error) {
			createOutput := input["create_user_output"].(map[string]interface{})
			email := createOutput["email"].(string)
			userID := createOutput["user_id"].(string)

			time.Sleep(100 * time.Millisecond)

			return map[string]interface{}{
				"user_id":    userID,
				"email":      email,
				"email_sent": true,
				"sent_at":    time.Now(),
				"email_type": "welcome",
			}, nil
		}).
		Sequential("validate_email", "check_duplicate", "create_user", "send_welcome_email")

	return workflow.Register()
}

// 工具函数
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr ||
		(len(str) > len(substr) && (str[:len(substr)] == substr ||
			str[len(str)-len(substr):] == substr ||
			containsInside(str, substr))))
}

func containsInside(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

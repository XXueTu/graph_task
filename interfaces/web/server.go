package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/XXueTu/graph_task/application"
)

// Server Web服务器
type Server struct {
	workflowService   *application.WorkflowService
	executionService  *application.ExecutionService
	retryService      *application.RetryService
	port             int
	mux              *http.ServeMux
}

// NewServer 创建Web服务器
func NewServer(
	workflowService *application.WorkflowService,
	executionService *application.ExecutionService,
	retryService *application.RetryService,
	port int,
) *Server {
	server := &Server{
		workflowService:  workflowService,
		executionService: executionService,
		retryService:     retryService,
		port:            port,
		mux:             http.NewServeMux(),
	}
	
	server.setupRoutes()
	return server
}

// Start 启动服务器
func (s *Server) Start() error {
	log.Printf("🌐 Starting web server on port %d", s.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), s.enableCORS(s.mux))
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 静态文件服务
	s.mux.Handle("/", http.FileServer(http.Dir("./web/static/")))
	
	// API路由
	s.mux.HandleFunc("/api/v1/workflows", s.handleWorkflows)
	s.mux.HandleFunc("/api/v1/workflows/", s.handleWorkflowByID)
	s.mux.HandleFunc("/api/v1/executions", s.handleExecutions)
	s.mux.HandleFunc("/api/v1/executions/", s.handleExecutionByID)
	s.mux.HandleFunc("/api/v1/retries", s.handleRetries)
	s.mux.HandleFunc("/api/v1/retries/", s.handleRetryByID)
	s.mux.HandleFunc("/api/v1/health", s.handleHealth)
}

// enableCORS 启用CORS
func (s *Server) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// handleWorkflows 处理工作流相关请求
func (s *Server) handleWorkflows(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.listWorkflows(w, r)
	case "POST":
		s.createWorkflow(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleWorkflowByID 处理特定工作流请求
func (s *Server) handleWorkflowByID(w http.ResponseWriter, r *http.Request) {
	workflowID := s.extractIDFromPath(r.URL.Path, "/api/v1/workflows/")
	if workflowID == "" {
		s.writeError(w, http.StatusBadRequest, "Invalid workflow ID")
		return
	}
	
	switch r.Method {
	case "GET":
		s.getWorkflow(w, r, workflowID)
	case "PUT":
		s.updateWorkflow(w, r, workflowID)
	case "DELETE":
		s.deleteWorkflow(w, r, workflowID)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleExecutions 处理执行相关请求
func (s *Server) handleExecutions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.listExecutions(w, r)
	case "POST":
		s.createExecution(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleExecutionByID 处理特定执行请求
func (s *Server) handleExecutionByID(w http.ResponseWriter, r *http.Request) {
	executionID := s.extractIDFromPath(r.URL.Path, "/api/v1/executions/")
	if executionID == "" {
		s.writeError(w, http.StatusBadRequest, "Invalid execution ID")
		return
	}
	
	// 处理子路径
	if strings.Contains(executionID, "/") {
		parts := strings.Split(executionID, "/")
		executionID = parts[0]
		subPath := parts[1]
		
		switch subPath {
		case "cancel":
			if r.Method == "POST" {
				s.cancelExecution(w, r, executionID)
			} else {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		case "status":
			if r.Method == "GET" {
				s.getExecutionStatus(w, r, executionID)
			} else {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		case "logs":
			if r.Method == "GET" {
				s.getExecutionLogs(w, r, executionID)
			} else {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		case "trace":
			if r.Method == "GET" {
				s.getExecutionTrace(w, r, executionID)
			} else {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		default:
			s.writeError(w, http.StatusNotFound, "Not found")
		}
		return
	}
	
	switch r.Method {
	case "GET":
		s.getExecution(w, r, executionID)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleRetries 处理重试相关请求
func (s *Server) handleRetries(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.listRetries(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleRetryByID 处理特定重试请求
func (s *Server) handleRetryByID(w http.ResponseWriter, r *http.Request) {
	retryID := s.extractIDFromPath(r.URL.Path, "/api/v1/retries/")
	if retryID == "" {
		s.writeError(w, http.StatusBadRequest, "Invalid retry ID")
		return
	}
	
	if strings.HasSuffix(retryID, "/retry") {
		retryID = strings.TrimSuffix(retryID, "/retry")
		if r.Method == "POST" {
			s.manualRetry(w, r, retryID)
		} else {
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
		return
	}
	
	switch r.Method {
	case "GET":
		s.getRetry(w, r, retryID)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleHealth 健康检查
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	}
	
	s.writeJSON(w, http.StatusOK, response)
}

// 工具方法
func (s *Server) extractIDFromPath(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	return strings.TrimPrefix(path, prefix)
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) parseIntParam(r *http.Request, param string, defaultValue int) int {
	value := r.URL.Query().Get(param)
	if value == "" {
		return defaultValue
	}
	
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	
	return intValue
}
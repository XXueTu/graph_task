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
	s.mux.Handle("/", http.FileServer(http.Dir("./interfaces/web/static/")))
	
	// API路由
	s.mux.HandleFunc("/api/v1/executions", s.handleExecutions)
	s.mux.HandleFunc("/api/v1/executions/", s.handleExecutionByID)
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


// handleExecutions 处理执行相关请求
func (s *Server) handleExecutions(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		s.listExecutions(w, r)
	} else {
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
		case "task-results":
			if r.Method == "GET" {
				s.getExecutionTaskResults(w, r, executionID)
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
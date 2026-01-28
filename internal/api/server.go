package api

import (
	"github.com/gin-gonic/gin"
	task "github.com/neyuki778/LLM-PDF-OCR/internal/task"
)

// Server 封装 Gin 引擎和依赖
type Server struct {
	router      *gin.Engine
	taskManager *task.TaskManager
}

// NewServer 创建 API 服务器实例
func NewServer(tm *task.TaskManager) *Server {
	r := gin.Default() // 自带 Logger 和 Recovery 中间件

	s := &Server{
		router:      r,
		taskManager: tm,
	}

	s.setupRoutes()
	return s
}

// setupRoutes 注册所有路由
func (s *Server) setupRoutes() {
	// API 路由
	api := s.router.Group("/api")
	{
		// Phase 4.1
		api.POST("/tasks", s.createTask)          // 上传 PDF，创建任务
		api.GET("/tasks/:id", s.getTask)          // 查询任务状态

		// Phase 4.2
		api.GET("/tasks/:id/result", s.getResult) // 下载结果
		api.DELETE("/tasks/:id", s.deleteTask)    // 删除任务
	}

	// 静态文件服务
	s.router.StaticFile("/", "./web/index.html")
	s.router.StaticFile("/style.css", "./web/style.css")
	s.router.Static("/dist", "./web/dist")
	s.router.Static("/output", "./output")   // 暴露分片后的 PDF 供 LLM-API 提供商拉取
}

// Run 启动 HTTP 服务
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

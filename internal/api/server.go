package api

import (
	"fmt"
	"log"

	"github.com/fengmingli/orchestrator/internal/api/handler"
	"github.com/fengmingli/orchestrator/internal/api/middleware"
	"github.com/fengmingli/orchestrator/internal/config"
	"github.com/fengmingli/orchestrator/internal/dal"
	"github.com/fengmingli/orchestrator/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Server API服务器
type Server struct {
	config   *config.Config
	db       *gorm.DB
	router   *gin.Engine
	services *service.Services
}

// NewServer 创建新的API服务器
func NewServer(cfg *config.Config) *Server {
	// 初始化数据库
	db, err := dal.InitMySQL(cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 初始化服务层
	services := service.NewServices(db)

	// 初始化路由
	router := gin.Default()

	server := &Server{
		config:   cfg,
		db:       db,
		router:   router,
		services: services,
	}

	server.setupRoutes()
	return server
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 中间件
	s.router.Use(middleware.CORS())
	s.router.Use(middleware.Logger())
	s.router.Use(middleware.Recovery())

	// API分组
	api := s.router.Group("/api/v1")

	// 步骤相关API
	stepHandler := handler.NewStepHandler(s.services.StepService)
	stepGroup := api.Group("/steps")
	{
		stepGroup.POST("", stepHandler.CreateStep)
		stepGroup.GET("", stepHandler.ListSteps)
		stepGroup.GET("/:id", stepHandler.GetStep)
		stepGroup.PUT("/:id", stepHandler.UpdateStep)
		stepGroup.DELETE("/:id", stepHandler.DeleteStep)
		stepGroup.POST("/:id/validate", stepHandler.ValidateStep)
	}

	// 模板相关API
	templateHandler := handler.NewTemplateHandler(s.services.TemplateService)
	templateGroup := api.Group("/templates")
	{
		templateGroup.POST("", templateHandler.CreateTemplate)
		templateGroup.GET("", templateHandler.ListTemplates)
		templateGroup.GET("/:id", templateHandler.GetTemplate)
		templateGroup.PUT("/:id", templateHandler.UpdateTemplate)
		templateGroup.DELETE("/:id", templateHandler.DeleteTemplate)
		templateGroup.GET("/:id/dag", templateHandler.GetTemplateDAG)
		templateGroup.POST("/:id/steps", templateHandler.AddStepToTemplate)
		templateGroup.DELETE("/:id/steps/:stepId", templateHandler.RemoveStepFromTemplate)
	}

	// 执行相关API
	executionHandler := handler.NewExecutionHandler(s.services.ExecutionService, s.services.OrchestratorService)
	executionGroup := api.Group("/executions")
	{
		executionGroup.POST("", executionHandler.CreateExecution)
		executionGroup.GET("", executionHandler.ListExecutions)
		executionGroup.GET("/:id", executionHandler.GetExecution)
		executionGroup.POST("/:id/start", executionHandler.StartExecution)
		executionGroup.POST("/:id/cancel", executionHandler.CancelExecution)
		executionGroup.GET("/:id/logs", executionHandler.GetExecutionLogs)
		executionGroup.GET("/:id/status", executionHandler.GetExecutionStatus)
	}

	// 健康检查
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}

// Run 启动服务器
func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port)
	return s.router.Run(addr)
}
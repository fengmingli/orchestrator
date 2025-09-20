package handler

import (
	"net/http"
	"strconv"

	"github.com/fengmingli/orchestrator/internal/model"
	"github.com/fengmingli/orchestrator/internal/service"
	"github.com/gin-gonic/gin"
)

// ExecutionHandler 执行处理器
type ExecutionHandler struct {
	executionService    *service.ExecutionService
	orchestratorService *service.OrchestratorService
}

// NewExecutionHandler 创建执行处理器
func NewExecutionHandler(executionService *service.ExecutionService, orchestratorService *service.OrchestratorService) *ExecutionHandler {
	return &ExecutionHandler{
		executionService:    executionService,
		orchestratorService: orchestratorService,
	}
}

// CreateExecution 创建执行
func (h *ExecutionHandler) CreateExecution(c *gin.Context) {
	var req model.ExecutionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	execution, err := h.executionService.CreateExecution(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建执行失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    201,
		"message": "执行创建成功",
		"data":    execution,
	})
}

// ListExecutions 获取执行列表
func (h *ExecutionHandler) ListExecutions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	templateID := c.Query("template_id")
	status := c.Query("status")

	executions, total, err := h.executionService.ListExecutions(page, size, templateID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取执行列表失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data": gin.H{
			"items": executions,
			"total": total,
			"page":  page,
			"size":  size,
		},
	})
}

// GetExecution 获取执行详情
func (h *ExecutionHandler) GetExecution(c *gin.Context) {
	id := c.Param("id")

	execution, err := h.executionService.GetExecution(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "执行不存在",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    execution,
	})
}

// StartExecution 启动执行
func (h *ExecutionHandler) StartExecution(c *gin.Context) {
	id := c.Param("id")

	// 异步启动执行
	go func() {
		h.orchestratorService.ExecuteWorkflow(id)
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "执行已启动",
	})
}

// CancelExecution 取消执行
func (h *ExecutionHandler) CancelExecution(c *gin.Context) {
	id := c.Param("id")

	err := h.executionService.CancelExecution(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "取消执行失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "执行已取消",
	})
}

// GetExecutionLogs 获取执行日志
func (h *ExecutionHandler) GetExecutionLogs(c *gin.Context) {
	id := c.Param("id")

	logs, err := h.executionService.GetExecutionLogs(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取执行日志失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    logs,
	})
}

// GetExecutionStatus 获取执行状态
func (h *ExecutionHandler) GetExecutionStatus(c *gin.Context) {
	id := c.Param("id")

	status, err := h.executionService.GetExecutionStatus(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取执行状态失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    status,
	})
}
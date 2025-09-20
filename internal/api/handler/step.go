package handler

import (
	"net/http"
	"strconv"

	"github.com/fengmingli/orchestrator/internal/model"
	"github.com/fengmingli/orchestrator/internal/service"
	"github.com/gin-gonic/gin"
)

// StepHandler 步骤处理器
type StepHandler struct {
	stepService *service.StepService
}

// NewStepHandler 创建步骤处理器
func NewStepHandler(stepService *service.StepService) *StepHandler {
	return &StepHandler{
		stepService: stepService,
	}
}

// CreateStep 创建步骤
func (h *StepHandler) CreateStep(c *gin.Context) {
	var req model.StepCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	step, err := h.stepService.CreateStep(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建步骤失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    201,
		"message": "步骤创建成功",
		"data":    step,
	})
}

// ListSteps 获取步骤列表
func (h *StepHandler) ListSteps(c *gin.Context) {
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	
	// 查询参数
	executorType := c.Query("executor_type")
	creatorEmail := c.Query("creator_email")

	steps, total, err := h.stepService.ListSteps(page, size, executorType, creatorEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取步骤列表失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data": gin.H{
			"items": steps,
			"total": total,
			"page":  page,
			"size":  size,
		},
	})
}

// GetStep 获取步骤详情
func (h *StepHandler) GetStep(c *gin.Context) {
	id := c.Param("id")

	step, err := h.stepService.GetStep(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "步骤不存在",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    step,
	})
}

// UpdateStep 更新步骤
func (h *StepHandler) UpdateStep(c *gin.Context) {
	id := c.Param("id")

	var req model.StepUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	step, err := h.stepService.UpdateStep(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新步骤失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
		"data":    step,
	})
}

// DeleteStep 删除步骤
func (h *StepHandler) DeleteStep(c *gin.Context) {
	id := c.Param("id")

	err := h.stepService.DeleteStep(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除步骤失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// ValidateStep 验证步骤参数
func (h *StepHandler) ValidateStep(c *gin.Context) {
	id := c.Param("id")

	isValid, err := h.stepService.ValidateStep(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "验证失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "验证完成",
		"data": gin.H{
			"valid": isValid,
		},
	})
}
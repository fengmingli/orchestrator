package handler

import (
	"net/http"
	"strconv"

	"github.com/fengmingli/orchestrator/internal/model"
	"github.com/fengmingli/orchestrator/internal/service"
	"github.com/gin-gonic/gin"
)

// TemplateHandler 模板处理器
type TemplateHandler struct {
	templateService *service.TemplateService
}

// NewTemplateHandler 创建模板处理器
func NewTemplateHandler(templateService *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{
		templateService: templateService,
	}
}

// CreateTemplate 创建模板
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	var req model.TemplateCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	template, err := h.templateService.CreateTemplate(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建模板失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    201,
		"message": "模板创建成功",
		"data":    template,
	})
}

// ListTemplates 获取模板列表
func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	creatorEmail := c.Query("creator_email")
	isActive := c.Query("is_active")

	templates, total, err := h.templateService.ListTemplates(page, size, creatorEmail, isActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取模板列表失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data": gin.H{
			"items": templates,
			"total": total,
			"page":  page,
			"size":  size,
		},
	})
}

// GetTemplate 获取模板详情
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	id := c.Param("id")

	template, err := h.templateService.GetTemplate(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "模板不存在",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    template,
	})
}

// UpdateTemplate 更新模板
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	id := c.Param("id")

	var req model.TemplateUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	template, err := h.templateService.UpdateTemplate(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新模板失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
		"data":    template,
	})
}

// DeleteTemplate 删除模板
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")

	err := h.templateService.DeleteTemplate(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除模板失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// GetTemplateDAG 获取模板DAG结构
func (h *TemplateHandler) GetTemplateDAG(c *gin.Context) {
	id := c.Param("id")

	dag, err := h.templateService.GetTemplateDAG(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取DAG结构失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    dag,
	})
}

// AddStepToTemplate 向模板添加步骤
func (h *TemplateHandler) AddStepToTemplate(c *gin.Context) {
	id := c.Param("id")

	var req model.TemplateStepCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	err := h.templateService.AddStepToTemplate(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "添加步骤失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "添加成功",
	})
}

// RemoveStepFromTemplate 从模板移除步骤
func (h *TemplateHandler) RemoveStepFromTemplate(c *gin.Context) {
	templateID := c.Param("id")
	stepID := c.Param("stepId")

	err := h.templateService.RemoveStepFromTemplate(templateID, stepID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "移除步骤失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "移除成功",
	})
}
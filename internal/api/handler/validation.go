package handler

import (
	"net/http"

	"github.com/fengmingli/orchestrator/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ValidationHandler 验证处理器
type ValidationHandler struct {
	validationService *service.ValidationService
}

// NewValidationHandler 创建验证处理器
func NewValidationHandler(db *gorm.DB) *ValidationHandler {
	return &ValidationHandler{
		validationService: service.NewValidationService(db),
	}
}

// ValidateDataConsistency 验证数据一致性
// @Summary 验证数据一致性
// @Description 检查数据库中的数据一致性，识别孤立数据
// @Tags validation
// @Accept json
// @Produce json
// @Success 200 {object} service.ValidationResult
// @Router /api/v1/validation/consistency [get]
func (h *ValidationHandler) ValidateDataConsistency(c *gin.Context) {
	result, err := h.validationService.ValidateDataConsistency()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "验证数据一致性失败",
			"details": err.Error(),
		})
		return
	}

	status := http.StatusOK
	if !result.IsValid {
		status = http.StatusBadRequest
	}

	c.JSON(status, gin.H{
		"data": result,
	})
}

// CleanupOrphanedData 清理孤立数据
// @Summary 清理孤立数据
// @Description 清理数据库中的孤立数据（谨慎使用）
// @Tags validation
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /api/v1/validation/cleanup [post]
func (h *ValidationHandler) CleanupOrphanedData(c *gin.Context) {
	err := h.validationService.CleanupOrphanedData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "清理孤立数据失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "孤立数据清理完成",
	})
}
package validator

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// InitValidators 初始化自定义验证器
func InitValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// 注册字母数字下划线横线验证器
		v.RegisterValidation("alphanum_underscore_dash", validateAlphanumUnderscoreDash)
	}
}

// validateAlphanumUnderscoreDash 验证字符串只包含字母、数字、下划线、横线
func validateAlphanumUnderscoreDash(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // 空值由required验证器处理
	}
	
	// 正则表达式：只允许大小写字母、数字、下划线、横线
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, value)
	return matched
}
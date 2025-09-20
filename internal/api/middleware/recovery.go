package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Recovery 恢复中间件
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logrus.WithFields(logrus.Fields{
			"path":      c.Request.URL.Path,
			"method":    c.Request.Method,
			"client_ip": c.ClientIP(),
			"panic":     recovered,
		}).Error("Panic recovered")

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "内部服务器错误",
		})
	})
}
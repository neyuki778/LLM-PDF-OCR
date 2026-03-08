package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 中间件处理 StatusServiceUnavailable
func (s *Server) requireAuthService() gin.HandlerFunc {
    return func(c *gin.Context) {
        if s.authService == nil {
            c.JSON(http.StatusServiceUnavailable, gin.H{
                "error": "auth service is not configured",
            })
            c.Abort() // 阻止继续执行后续 handler
            return
        }
        c.Next() // 继续
    }
}
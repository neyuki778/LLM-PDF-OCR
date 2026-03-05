package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	auth "github.com/neyuki778/LLM-PDF-OCR/internal/auth"
)

// logout 处理 POST /api/auth/logout
func (s *Server) logout(c *gin.Context) {
	if s.authService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "auth service is not configured",
		})
		return
	}

	refreshToken, _ := c.Cookie(refreshTokenCookieName)
	if err := s.authService.Logout(c.Request.Context(), refreshToken); err != nil {
		if errors.Is(err, auth.ErrInvalidRefreshToken) {
			// 保持登出语义幂等：即使 token 无效，也继续清 cookie 并返回成功。
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to logout",
			})
			return
		}
	}

	clearAuthCookies(c, s.authCookieSecure)
	c.JSON(http.StatusOK, gin.H{
		"message": "logged out",
	})
}

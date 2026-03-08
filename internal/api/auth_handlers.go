package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	auth "github.com/neyuki778/LLM-PDF-OCR/internal/auth"
)

// logout 处理 POST /api/auth/logout
func (s *Server) logout(c *gin.Context) {
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

// login 处理 POST /api/auth/login
func (s *Server) login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	result, err := s.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			clearAuthCookies(c, s.authCookieSecure)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid email or password",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to login",
		})
		return
	}

	setAuthCookies(
		c,
		s.authCookieSecure,
		result.AccessToken,
		result.AccessTokenExpires,
		result.RefreshToken,
		result.RefreshExpires,
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "login successful",
		"user": gin.H{
			"id":    result.User.ID,
			"email": result.User.Email,
		},
	})
}

func (s *Server) refresh(c *gin.Context) {
	refreshToken, _ := c.Cookie(refreshTokenCookieName)
	result, err := s.authService.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidRefreshToken) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid refresh token",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to refresh",
		})
		return
	}

	setAuthCookies(
		c,
		s.authCookieSecure,
		result.AccessToken,
		result.AccessTokenExpires,
		result.RefreshToken,
		result.RefreshExpires,
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "refresh successful",
	})
}

func (s *Server) register(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	user, err := s.authService.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUserExists):
			c.JSON(http.StatusConflict, gin.H{
				"error": "email already registered",
			})
			return
		case errors.Is(err, auth.ErrInvalidEmail), errors.Is(err, auth.ErrPasswordTooShort):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to register",
			})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "register successful",
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
		},
	})
}

// me 处理 GET /api/auth/me
func (s *Server) me(c *gin.Context) {
	accessToken, _ := c.Cookie(accessTokenCookieName)
	user, err := s.authService.Me(c.Request.Context(), accessToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidAccessToken) {
			clearAuthCookies(c, s.authCookieSecure)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid access token",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to load current user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
		},
	})
}

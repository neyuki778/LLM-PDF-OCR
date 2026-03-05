package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	accessTokenCookieName  = "access_token"
	refreshTokenCookieName = "refresh_token"
)

func clearAuthCookies(c *gin.Context, secure bool) {
	// 告诉浏览器删除cookie的标准做法
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(accessTokenCookieName, "", -1, "/", "", secure, true)

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(refreshTokenCookieName, "", -1, "/", "", secure, true)
}

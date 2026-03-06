package api

import (
	"net/http"
	"time"

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

func setAuthCookies(
	c *gin.Context,
	secure bool,
	accessToken string,
	accessExpires time.Time,
	refreshToken string,
	refreshExpires time.Time,
) {
	setTokenCookie(c, accessTokenCookieName, accessToken, accessExpires, secure)
	setTokenCookie(c, refreshTokenCookieName, refreshToken, refreshExpires, secure)
}

func setTokenCookie(c *gin.Context, name, value string, expiresAt time.Time, secure bool) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge <= 0 {
		maxAge = 1
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, maxAge, "/", "", secure, true)
}

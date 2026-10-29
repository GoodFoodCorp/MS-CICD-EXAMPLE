package controllers

import (
	"net/http"
	"os"

	"auth-service/internal/tokens"

	"github.com/gin-gonic/gin"
)

const (
	accessCookie  = "auth_token"
	refreshCookie = "refresh_token"
	// refreshCookiePath scopes the refresh token to the auth endpoints that
	// need it: /refresh and /logout (which must be able to revoke it).
	refreshCookiePath = "/api/auth"
	// legacyRefreshCookiePath is where older versions set the cookie; it is
	// cleared on logout so no stale copy survives.
	legacyRefreshCookiePath = "/api/auth/refresh"
)

// secureCookies reports whether session cookies get the Secure flag:
// COOKIE_SECURE wins when set, otherwise production or an https front.
func secureCookies() bool {
	if v := os.Getenv("COOKIE_SECURE"); v != "" {
		return v == "true"
	}
	return os.Getenv("ENV") == "production" || isSecureCookie()
}

// setSessionCookies stores the token pair in HttpOnly, SameSite=Lax cookies.
func setSessionCookies(c *gin.Context, accessToken, refreshToken string) {
	secure := secureCookies()
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(accessCookie, accessToken, int(tokens.AccessTTL.Seconds()), "/", "", secure, true)
	c.SetCookie(refreshCookie, refreshToken, int(tokens.RefreshTTL.Seconds()), refreshCookiePath, "", secure, true)
}

// clearSessionCookies expires every session cookie, legacy path included.
func clearSessionCookies(c *gin.Context) {
	secure := secureCookies()
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(accessCookie, "", -1, "/", "", secure, true)
	c.SetCookie(refreshCookie, "", -1, refreshCookiePath, "", secure, true)
	c.SetCookie(refreshCookie, "", -1, legacyRefreshCookiePath, "", secure, true)
}

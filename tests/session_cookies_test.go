package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"


	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func cookieByName(w *httptest.ResponseRecorder, name string) []*http.Cookie {
	var out []*http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			out = append(out, c)
		}
	}
	return out
}

func TestLogin_SessionCookiesAreHardened(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "true")
	srv := new(MockAuthService)
	srv.On("LoginUser", mock.AnythingOfType("*models.LoginRequest")).Return("at", "rt", nil).Once()

	w := doJSON(SetupAuthRouter(srv), http.MethodPost, "/api/auth/login", `{"email":"a@b.fr","password":"Password1"}`)

	assert.Equal(t, http.StatusOK, w.Code)
	refresh := cookieByName(w, "refresh_token")
	assert.Len(t, refresh, 1)
	// /api/auth (not /api/auth/refresh): logout must receive it to revoke it.
	assert.Equal(t, "/api/auth", refresh[0].Path)
	for _, c := range w.Result().Cookies() {
		assert.True(t, c.HttpOnly, c.Name)
		assert.True(t, c.Secure, c.Name)
		assert.Equal(t, http.SameSiteLaxMode, c.SameSite, c.Name)
	}
}

func TestLogout_ClearsLegacyRefreshCookieToo(t *testing.T) {
	srv := new(MockAuthService)
	srv.On("LogoutUser", "rt").Return(nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "rt"})
	SetupAuthRouter(srv).ServeHTTP(w, req)

	paths := map[string]bool{}
	for _, c := range cookieByName(w, "refresh_token") {
		paths[c.Path] = c.MaxAge < 0
	}
	assert.True(t, paths["/api/auth"])
	assert.True(t, paths["/api/auth/refresh"])
	srv.AssertExpectations(t)
}

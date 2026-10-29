package tests

import (
	"auth-service/internal/services"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotifyUserServiceErasureCallsTheInternalRoute(t *testing.T) {
	var method, path, key string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, key = r.Method, r.URL.Path, r.Header.Get("X-Service-Key")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	t.Setenv("USER_SERVICE_URL", srv.URL)
	t.Setenv("SERVICE_API_KEY", "svc")

	services.NotifyUserServiceErasure("U1")

	assert.Equal(t, http.MethodDelete, method)
	assert.Equal(t, "/internal/profiles/U1", path)
	assert.Equal(t, "svc", key)
}

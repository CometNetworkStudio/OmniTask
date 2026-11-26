package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func newCSRFRouter(env string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	conf := viper.New()
	conf.Set("env", env)
	r := gin.New()
	r.Use(CSRF(conf))
	r.POST("/api/v1/tasks", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })
	r.POST("/api/v1/auth/login", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })
	r.GET("/api/v1/me", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })
	return r
}

func doCSRF(r *gin.Engine, method, path, origin, header, cookie string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if header != "" {
		req.Header.Set("X-CSRF-Token", header)
	}
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: "omnitask_csrf", Value: cookie})
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCSRF(t *testing.T) {
	r := newCSRFRouter("prod")
	if w := doCSRF(r, http.MethodGet, "/api/v1/me", "", "", ""); w.Code != http.StatusOK {
		t.Fatalf("GET should pass, got %d", w.Code)
	}
	if w := doCSRF(r, http.MethodPost, "/api/v1/tasks", "", "", ""); w.Code != http.StatusForbidden {
		t.Fatalf("missing token should fail, got %d", w.Code)
	}
	if w := doCSRF(r, http.MethodPost, "/api/v1/tasks", "", "abc", "abc"); w.Code != http.StatusOK {
		t.Fatalf("matching token should pass, got %d", w.Code)
	}
	if w := doCSRF(r, http.MethodPost, "/api/v1/tasks", "https://evil.example", "abc", "abc"); w.Code != http.StatusForbidden {
		t.Fatalf("cross origin should fail, got %d", w.Code)
	}
	if w := doCSRF(r, http.MethodPost, "/api/v1/auth/login", "", "", ""); w.Code != http.StatusOK {
		t.Fatalf("login should skip token check, got %d", w.Code)
	}
}

func TestCSRFDevAllowsLocalhostOrigin(t *testing.T) {
	r := newCSRFRouter("dev")
	if w := doCSRF(r, http.MethodPost, "/api/v1/auth/login", "http://localhost:5173", "", ""); w.Code != http.StatusOK {
		t.Fatalf("dev localhost origin should pass, got %d", w.Code)
	}
}

func TestCSRFAllowsLoopbackProxyOrigin(t *testing.T) {
	r := newCSRFRouter("prod")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("loopback proxy origin should pass, got %d", w.Code)
	}
}

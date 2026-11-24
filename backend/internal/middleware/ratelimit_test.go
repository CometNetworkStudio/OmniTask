package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestIPRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(IPRateLimit(2, time.Minute))
	r.POST("/login", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	do := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "203.0.113.9:12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	if w := do(); w.Code != http.StatusOK {
		t.Fatalf("first request: %d", w.Code)
	}
	if w := do(); w.Code != http.StatusOK {
		t.Fatalf("second request: %d", w.Code)
	}
	w := do()
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("third request should be limited, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After header")
	}
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Code != 1019 {
		t.Fatalf("unexpected body: %s err=%v", w.Body.String(), err)
	}
}

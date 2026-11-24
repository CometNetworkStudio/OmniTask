package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const defaultCSP = "default-src 'self'; base-uri 'self'; object-src 'none'; " +
	"frame-ancestors 'none'; img-src 'self' data: blob:; font-src 'self' data:; " +
	"style-src 'self' 'unsafe-inline'; script-src 'self'; " +
	"connect-src 'self' ws: wss:; form-action 'self'"

// swaggerUI 需要内联脚本与样式；仅对 /swagger 放宽，不影响主应用。
const swaggerCSP = "default-src 'self'; img-src 'self' data:; font-src 'self' data:; " +
	"style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; " +
	"connect-src 'self'; frame-ancestors 'none'"

// SecurityHeaders 下发基础安全响应头；HSTS 仅在确认 HTTPS 时下发。
func SecurityHeaders() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		h := ctx.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		path := ctx.Request.URL.Path
		if strings.HasPrefix(path, "/swagger") {
			h.Set("Content-Security-Policy", swaggerCSP)
		} else {
			h.Set("Content-Security-Policy", defaultCSP)
		}
		if strings.HasPrefix(path, "/api/") {
			h.Set("Cache-Control", "no-store")
		}
		if ctx.Request.TLS != nil || strings.EqualFold(ctx.GetHeader("X-Forwarded-Proto"), "https") {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		ctx.Next()
	}
}

package middleware

import (
	"net/url"
	"strings"
	"time"

	"backend/pkg/log"
	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/duke-git/lancet/v2/random"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// 查询串里可能承载凭据或令牌（WebSocket 握手可用 accessToken），日志必须脱敏。
var sensitiveQueryKeys = map[string]struct{}{
	"accesstoken":  {},
	"access_token": {},
	"token":        {},
	"password":     {},
	"passwd":       {},
	"secret":       {},
	"key":          {},
	"credential":   {},
	"cookie":       {},
	"session":      {},
	"code":         {},
	"auth":         {},
}

func redactQuery(raw string) string {
	if raw == "" {
		return ""
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "<redacted>"
	}
	for key := range values {
		if _, sensitive := sensitiveQueryKeys[strings.ToLower(key)]; sensitive {
			for i := range values[key] {
				values[key][i] = "***"
			}
		}
	}
	return values.Encode()
}

// RequestLogMiddleware 只记录方法、路径与脱敏后的查询串。
// 请求体可能含凭据明文，响应体可能含登录 Token，均不得写入日志。
func RequestLogMiddleware(logger *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		uuid, err := random.UUIdV4()
		if err != nil {
			ctx.Next()
			return
		}
		trace := cryptor.Md5String(uuid)
		logger.WithValue(ctx, zap.String("trace", trace))
		logger.WithValue(ctx, zap.String("request_method", ctx.Request.Method))
		logger.WithValue(ctx, zap.String("request_path", ctx.Request.URL.Path))
		logger.WithValue(ctx, zap.String("request_query", redactQuery(ctx.Request.URL.RawQuery)))
		logger.WithValue(ctx, zap.Int64("request_size", ctx.Request.ContentLength))
		logger.WithValue(ctx, zap.String("request_content_type", ctx.ContentType()))
		logger.WithContext(ctx).Info("Request")
		ctx.Next()
	}
}

// ResponseLogMiddleware 只记录状态码、响应大小与耗时，不记录响应正文。
func ResponseLogMiddleware(logger *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startTime := time.Now()
		ctx.Next()
		logger.WithContext(ctx).Info("Response",
			zap.Int("status", ctx.Writer.Status()),
			zap.Int("size", ctx.Writer.Size()),
			zap.String("time", time.Since(startTime).String()),
		)
	}
}

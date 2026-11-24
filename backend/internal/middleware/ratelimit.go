package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	v1 "backend/api/v1"
	"github.com/gin-gonic/gin"
)

type rateBucket struct {
	count int
	reset time.Time
}

// IPRateLimit 对单个 IP 做固定窗口限流，用于登录/注册等无鉴权接口。
// limit <= 0 表示关闭。计数保存在进程内，lite 单实例足够；多实例部署需改用共享存储。
func IPRateLimit(limit int, window time.Duration) gin.HandlerFunc {
	if limit <= 0 || window <= 0 {
		return func(ctx *gin.Context) { ctx.Next() }
	}
	var (
		mu      sync.Mutex
		buckets = map[string]*rateBucket{}
	)
	return func(ctx *gin.Context) {
		now := time.Now()
		key := ctx.ClientIP()

		mu.Lock()
		for ip, bucket := range buckets {
			if now.After(bucket.reset) {
				delete(buckets, ip)
			}
		}
		bucket := buckets[key]
		if bucket == nil || now.After(bucket.reset) {
			bucket = &rateBucket{reset: now.Add(window)}
			buckets[key] = bucket
		}
		bucket.count++
		allowed := bucket.count <= limit
		retryAfter := int(time.Until(bucket.reset).Seconds()) + 1
		mu.Unlock()

		if !allowed {
			ctx.Header("Retry-After", strconv.Itoa(retryAfter))
			v1.HandleError(ctx, http.StatusTooManyRequests, v1.ErrTooManyRequests, nil)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

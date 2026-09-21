package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware 仅对配置允许的 Origin 放开跨域（默认空=同源，不下发 CORS 头）。
func CORSMiddleware(origins []string) gin.HandlerFunc {
	allowed := map[string]bool{}
	allowAll := false
	for _, origin := range origins {
		if origin == "*" {
			allowAll = true
		}
		if origin != "" {
			allowed[origin] = true
		}
	}
	permitted := func(origin string) bool {
		return origin != "" && (allowAll || allowed[origin])
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if permitted(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if c.Request.Method == http.MethodOptions {
			if permitted(origin) {
				c.Header("Access-Control-Allow-Methods", c.GetHeader("Access-Control-Request-Method"))
				c.Header("Access-Control-Allow-Headers", c.GetHeader("Access-Control-Request-Headers"))
				c.Header("Access-Control-Max-Age", "7200")
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

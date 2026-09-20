package middleware

import (
	"net/http"

	v1 "backend/api/v1"
	"backend/pkg/jwt"
	"backend/pkg/log"
	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
)

// Authorize 基于 Casbin 对 (role, route, method) 鉴权；须在 StrictAuth 之后。
func Authorize(e *casbin.Enforcer, logger *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		role := "user"
		if v, ok := ctx.Get("claims"); ok {
			if claims, ok := v.(*jwt.MyCustomClaims); ok && claims.Role != "" {
				role = claims.Role
			}
		}
		path := ctx.FullPath()
		if path == "" {
			path = ctx.Request.URL.Path
		}
		allowed, err := e.Enforce(role, path, ctx.Request.Method)
		if err != nil {
			logger.WithContext(ctx).Error("casbin enforce error")
			v1.HandleError(ctx, http.StatusInternalServerError, v1.ErrInternalServerError, nil)
			ctx.Abort()
			return
		}
		if !allowed {
			v1.HandleError(ctx, http.StatusForbidden, v1.ErrForbidden, nil)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

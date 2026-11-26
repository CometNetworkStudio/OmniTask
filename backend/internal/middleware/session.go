package middleware

import (
	"errors"
	"net/http"

	v1 "backend/api/v1"
	"backend/internal/service"
	"backend/pkg/authcookie"
	"backend/pkg/log"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// SessionAuth 从 HttpOnly Cookie 读取服务端会话，校验通过后把用户和会话放进上下文。
func SessionAuth(sessions *service.SessionService, conf *viper.Viper, logger *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token, err := ctx.Cookie(authcookie.SessionCookie)
		if err != nil || token == "" {
			unauthorized(ctx, conf, "")
			ctx.Abort()
			return
		}
		user, session, err := sessions.Validate(ctx.Request.Context(), token)
		if err != nil {
			if !errors.Is(err, v1.ErrUnauthorized) {
				logger.WithContext(ctx).Error("validate session failed: " + err.Error())
				v1.HandleError(ctx, http.StatusInternalServerError, v1.ErrInternalServerError, nil)
				ctx.Abort()
				return
			}
			unauthorized(ctx, conf, token)
			ctx.Abort()
			return
		}
		ctx.Set(authcookie.CtxUserKey, user)
		ctx.Set(authcookie.CtxSessionKey, session)
		ctx.Next()
	}
}

func unauthorized(ctx *gin.Context, conf *viper.Viper, token string) {
	if token != "" {
		opts := authcookie.OptionsFrom(conf, ctx.Request)
		authcookie.ClearSession(ctx.Writer, opts)
		authcookie.ClearCSRF(ctx.Writer, opts)
	}
	v1.HandleError(ctx, http.StatusUnauthorized, v1.ErrUnauthorized, nil)
}

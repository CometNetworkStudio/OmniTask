package handler

import (
	"backend/internal/model"
	"backend/pkg/authcookie"
	"backend/pkg/log"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	logger *log.Logger
}

func NewHandler(
	logger *log.Logger,
) *Handler {
	return &Handler{
		logger: logger,
	}
}

func GetUserFromCtx(ctx *gin.Context) *model.User {
	v, exists := ctx.Get(authcookie.CtxUserKey)
	if !exists {
		return nil
	}
	user, _ := v.(*model.User)
	return user
}

func GetSessionFromCtx(ctx *gin.Context) *model.Session {
	v, exists := ctx.Get(authcookie.CtxSessionKey)
	if !exists {
		return nil
	}
	session, _ := v.(*model.Session)
	return session
}

func GetUserIdFromCtx(ctx *gin.Context) string {
	if user := GetUserFromCtx(ctx); user != nil {
		return user.ID
	}
	return ""
}

func GetRoleFromCtx(ctx *gin.Context) string {
	if user := GetUserFromCtx(ctx); user != nil {
		return user.Role
	}
	return ""
}

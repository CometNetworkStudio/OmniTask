package handler

import (
	"net/http"
	"strconv"

	v1 "backend/api/v1"
	"backend/internal/service"
	"backend/pkg/authcookie"
	"backend/pkg/log"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type UserHandler struct {
	svc      *service.UserService
	sessions *service.SessionService
	conf     *viper.Viper
	logger   *log.Logger
}

func NewUserHandler(svc *service.UserService, sessions *service.SessionService, conf *viper.Viper, logger *log.Logger) *UserHandler {
	return &UserHandler{svc: svc, sessions: sessions, conf: conf, logger: logger}
}

func (h *UserHandler) Me(ctx *gin.Context) {
	resp, err := h.svc.Me(ctx, GetUserIdFromCtx(ctx))
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *UserHandler) Preferences(ctx *gin.Context) {
	resp, err := h.svc.Preferences(ctx, GetUserIdFromCtx(ctx))
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *UserHandler) UpdatePreferences(ctx *gin.Context) {
	var req v1.UpdatePreferencesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.UpdatePreferences(ctx, GetUserIdFromCtx(ctx), &req)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *UserHandler) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "0"))
	list, total, err := h.svc.List(ctx, page, size)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, gin.H{"list": list, "total": total})
}

func (h *UserHandler) UpdateRole(ctx *gin.Context) {
	var req v1.UpdateRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.UpdateRole(ctx, ctx.Param("id"), req.Role)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *UserHandler) SetEnabled(ctx *gin.Context) {
	var req v1.SetEnabledRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.SetEnabled(ctx, ctx.Param("id"), req.Enabled)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

// MySessions 返回当前用户仍有效的登录会话。
func (h *UserHandler) MySessions(ctx *gin.Context) {
	current := ""
	if session := GetSessionFromCtx(ctx); session != nil {
		current = session.ID
	}
	list, err := h.sessions.ListUser(ctx, GetUserIdFromCtx(ctx), current)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, list)
}

func (h *UserHandler) RevokeMySession(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := h.sessions.RevokeOwned(ctx, GetUserIdFromCtx(ctx), id); err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	if session := GetSessionFromCtx(ctx); session != nil && session.ID == id {
		opts := authcookie.OptionsFrom(h.conf, ctx.Request)
		authcookie.ClearSession(ctx.Writer, opts)
		authcookie.ClearCSRF(ctx.Writer, opts)
	}
	v1.HandleSuccess(ctx, nil)
}

// RevokeOtherSessions 退出除当前设备外的所有会话。
func (h *UserHandler) RevokeOtherSessions(ctx *gin.Context) {
	current := ""
	if session := GetSessionFromCtx(ctx); session != nil {
		current = session.ID
	}
	if err := h.sessions.RevokeOthers(ctx, GetUserIdFromCtx(ctx), current); err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

func (h *UserHandler) AdminOverview(ctx *gin.Context) {
	data, err := h.sessions.Overview(ctx)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, data)
}

func (h *UserHandler) AdminSessions(ctx *gin.Context) {
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "100"))
	current := ""
	if session := GetSessionFromCtx(ctx); session != nil {
		current = session.ID
	}
	list, err := h.sessions.ListAll(ctx, current, limit)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, list)
}

func (h *UserHandler) AdminRevokeSession(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := h.sessions.RevokeByID(ctx, id); err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	if session := GetSessionFromCtx(ctx); session != nil && session.ID == id {
		opts := authcookie.OptionsFrom(h.conf, ctx.Request)
		authcookie.ClearSession(ctx.Writer, opts)
		authcookie.ClearCSRF(ctx.Writer, opts)
	}
	v1.HandleSuccess(ctx, nil)
}

func (h *UserHandler) AdminRevokeUserSessions(ctx *gin.Context) {
	if err := h.sessions.RevokeUser(ctx, ctx.Param("id")); err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

func (h *UserHandler) AdminLoginAttempts(ctx *gin.Context) {
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "50"))
	list, err := h.sessions.ListAttempts(ctx, limit)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, list)
}

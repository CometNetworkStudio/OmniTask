package handler

import (
	"net/http"
	"strconv"

	v1 "backend/api/v1"
	"backend/internal/service"
	"backend/pkg/log"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	svc    *service.UserService
	logger *log.Logger
}

func NewUserHandler(svc *service.UserService, logger *log.Logger) *UserHandler {
	return &UserHandler{svc: svc, logger: logger}
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

package handler

import (
	v1 "backend/api/v1"
	"backend/internal/service"
	"backend/pkg/log"
	"github.com/gin-gonic/gin"
)

type PlatformHandler struct {
	svc    *service.DiscoverService
	logger *log.Logger
}

func NewPlatformHandler(svc *service.DiscoverService, logger *log.Logger) *PlatformHandler {
	return &PlatformHandler{svc: svc, logger: logger}
}

func (h *PlatformHandler) List(ctx *gin.Context) {
	list, err := h.svc.Platforms(ctx)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, list)
}

func (h *PlatformHandler) Items(ctx *gin.Context) {
	resp, err := h.svc.Items(ctx, GetUserIdFromCtx(ctx), ctx.Param("id"), ctx.Query("credential_id"))
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *PlatformHandler) Refresh(ctx *gin.Context) {
	var body struct {
		CredentialID string `json:"credential_id"`
	}
	_ = ctx.ShouldBindJSON(&body)
	if body.CredentialID == "" {
		body.CredentialID = ctx.Query("credential_id")
	}
	resp, err := h.svc.Refresh(ctx, GetUserIdFromCtx(ctx), ctx.Param("id"), body.CredentialID)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

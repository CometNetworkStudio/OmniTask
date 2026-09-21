package handler

import (
	"net/http"

	v1 "backend/api/v1"
	"backend/internal/service"
	"backend/pkg/log"
	"github.com/gin-gonic/gin"
)

type ScriptHandler struct {
	svc    *service.ScriptService
	logger *log.Logger
}

func NewScriptHandler(svc *service.ScriptService, logger *log.Logger) *ScriptHandler {
	return &ScriptHandler{svc: svc, logger: logger}
}

func (h *ScriptHandler) List(ctx *gin.Context) {
	list, err := h.svc.List(ctx)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, list)
}

func (h *ScriptHandler) Install(ctx *gin.Context) {
	var req v1.ScriptInstallRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	jobID, err := h.svc.InstallAsync(req.URL, req.Ref)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, gin.H{"job_id": jobID})
}

func (h *ScriptHandler) InstallJob(ctx *gin.Context) {
	job, ok := h.svc.Job(ctx.Param("job"))
	if !ok {
		v1.HandleError(ctx, http.StatusNotFound, v1.ErrNotFound, nil)
		return
	}
	resp := gin.H{
		"status":  job.Status,
		"stage":   job.Stage,
		"percent": job.Percent,
		"message": job.Message,
		"error":   job.Error,
	}
	if job.Script != nil {
		resp["script"] = job.Script
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *ScriptHandler) Toggle(ctx *gin.Context) {
	var req v1.ScriptToggleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.SetEnabled(ctx, ctx.Param("id"), req.Enabled); err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, gin.H{"id": ctx.Param("id"), "enabled": req.Enabled})
}

func (h *ScriptHandler) Uninstall(ctx *gin.Context) {
	if err := h.svc.Uninstall(ctx, ctx.Param("id")); err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, gin.H{"id": ctx.Param("id")})
}

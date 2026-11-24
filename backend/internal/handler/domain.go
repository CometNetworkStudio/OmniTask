package handler

import (
	"errors"
	"net/http"
	"strconv"

	v1 "backend/api/v1"
	"backend/internal/service"
	"backend/pkg/log"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc    *service.AuthService
	logger *log.Logger
}

func NewAuthHandler(svc *service.AuthService, logger *log.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, logger: logger}
}

func (h *AuthHandler) Register(ctx *gin.Context) {
	var req v1.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.Register(ctx, &req)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *AuthHandler) Login(ctx *gin.Context) {
	var req v1.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.Login(ctx, &req)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

type CredentialHandler struct {
	svc    *service.CredentialService
	logger *log.Logger
}

func NewCredentialHandler(svc *service.CredentialService, logger *log.Logger) *CredentialHandler {
	return &CredentialHandler{svc: svc, logger: logger}
}

func (h *CredentialHandler) Create(ctx *gin.Context) {
	var req v1.CredentialCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.Create(ctx, GetUserIdFromCtx(ctx), &req)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *CredentialHandler) Update(ctx *gin.Context) {
	var req v1.CredentialUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.Update(ctx, GetUserIdFromCtx(ctx), ctx.Param("id"), &req)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *CredentialHandler) Archive(ctx *gin.Context) {
	if err := h.svc.Archive(ctx, GetUserIdFromCtx(ctx), ctx.Param("id")); err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

func (h *CredentialHandler) List(ctx *gin.Context) {
	list, err := h.svc.List(ctx, GetUserIdFromCtx(ctx))
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, list)
}

func (h *CredentialHandler) AccessLogs(ctx *gin.Context) {
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "50"))
	list, err := h.svc.ListAccessLogs(ctx, GetUserIdFromCtx(ctx), limit)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, list)
}

type TaskHandler struct {
	svc    *service.TaskService
	orch   *service.Orchestrator
	logger *log.Logger
}

func NewTaskHandler(svc *service.TaskService, orch *service.Orchestrator, logger *log.Logger) *TaskHandler {
	return &TaskHandler{svc: svc, orch: orch, logger: logger}
}

func (h *TaskHandler) Create(ctx *gin.Context) {
	var req v1.TaskCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.Create(ctx, GetUserIdFromCtx(ctx), &req)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, gin.H{"error": err.Error()})
		return
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *TaskHandler) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "0"))
	status := ctx.Query("status")
	platform := ctx.Query("platform")
	list, total, err := h.svc.List(ctx, GetUserIdFromCtx(ctx), status, platform, page, size)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, gin.H{"list": list, "total": total, "page": page, "size": size})
}

func (h *TaskHandler) Get(ctx *gin.Context) {
	resp, err := h.svc.Get(ctx, GetUserIdFromCtx(ctx), ctx.Param("id"))
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *TaskHandler) Events(ctx *gin.Context) {
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "1000"))
	list, err := h.svc.Events(ctx, GetUserIdFromCtx(ctx), ctx.Param("id"), limit)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, list)
}

func (h *TaskHandler) Cancel(ctx *gin.Context) {
	if err := h.orch.CancelTask(ctx, GetUserIdFromCtx(ctx), ctx.Param("id")); err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, gin.H{"id": ctx.Param("id"), "status": "canceled"})
}

func (h *TaskHandler) Retry(ctx *gin.Context) {
	resp, err := h.svc.Retry(ctx, GetUserIdFromCtx(ctx), ctx.Param("id"))
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *TaskHandler) Delete(ctx *gin.Context) {
	if err := h.svc.Delete(ctx, GetUserIdFromCtx(ctx), ctx.Param("id")); err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, gin.H{"id": ctx.Param("id")})
}

type QuestionHandler struct {
	svc    *service.QuestionService
	logger *log.Logger
}

func NewQuestionHandler(svc *service.QuestionService, logger *log.Logger) *QuestionHandler {
	return &QuestionHandler{svc: svc, logger: logger}
}

func (h *QuestionHandler) Lookup(ctx *gin.Context) {
	platform := ctx.Query("platform")
	key := ctx.Query("question_key")
	if platform == "" || key == "" {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	resp, err := h.svc.Lookup(ctx, platform, key)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *QuestionHandler) Save(ctx *gin.Context) {
	var req v1.QuestionSaveRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.Save(ctx, &req)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *QuestionHandler) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "20"))
	list, total, err := h.svc.List(ctx, ctx.Query("platform"), page, size)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, gin.H{"list": list, "total": total, "page": page, "size": size})
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, v1.ErrUnauthorized), errors.Is(err, v1.ErrInvalidCredentials):
		return http.StatusUnauthorized
	case errors.Is(err, v1.ErrNotFound), errors.Is(err, v1.ErrTaskNotFound), errors.Is(err, v1.ErrCredentialNotFound):
		return http.StatusNotFound
	case errors.Is(err, v1.ErrUserAlreadyExists), errors.Is(err, v1.ErrUnsupportedPlatform),
		errors.Is(err, v1.ErrBadRequest), errors.Is(err, v1.ErrCredentialEncrypt),
		errors.Is(err, v1.ErrTaskNotCancelable), errors.Is(err, v1.ErrRegistrationClosed),
		errors.Is(err, v1.ErrInvalidTaskParams), errors.Is(err, v1.ErrTaskLimit),
		errors.Is(err, v1.ErrQueueFull):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

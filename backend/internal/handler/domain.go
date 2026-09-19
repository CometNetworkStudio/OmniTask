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

func (h *CredentialHandler) Upsert(ctx *gin.Context) {
	var req v1.CredentialUpsertRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.Upsert(ctx, GetUserIdFromCtx(ctx), &req)
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, resp)
}

func (h *CredentialHandler) List(ctx *gin.Context) {
	list, err := h.svc.List(ctx, GetUserIdFromCtx(ctx))
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	v1.HandleSuccess(ctx, list)
}

type TaskHandler struct {
	svc    *service.TaskService
	logger *log.Logger
}

func NewTaskHandler(svc *service.TaskService, logger *log.Logger) *TaskHandler {
	return &TaskHandler{svc: svc, logger: logger}
}

func (h *TaskHandler) Create(ctx *gin.Context) {
	var req v1.TaskCreateRequest
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

func (h *TaskHandler) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "20"))
	list, total, err := h.svc.List(ctx, GetUserIdFromCtx(ctx), page, size)
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
		errors.Is(err, v1.ErrBadRequest), errors.Is(err, v1.ErrCredentialEncrypt):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

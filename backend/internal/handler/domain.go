package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	v1 "backend/api/v1"
	"backend/internal/service"
	"backend/pkg/authcookie"
	"backend/pkg/log"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type AuthHandler struct {
	svc    *service.AuthService
	conf   *viper.Viper
	logger *log.Logger
}

func NewAuthHandler(svc *service.AuthService, conf *viper.Viper, logger *log.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, conf: conf, logger: logger}
}

func (h *AuthHandler) Register(ctx *gin.Context) {
	var req v1.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, token, err := h.svc.Register(ctx, &req, ctx.ClientIP(), ctx.Request.UserAgent())
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	h.writeAuthCookies(ctx, token)
	v1.HandleSuccess(ctx, resp)
}

func (h *AuthHandler) Login(ctx *gin.Context) {
	var req v1.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, token, err := h.svc.Login(ctx, &req, ctx.ClientIP(), ctx.Request.UserAgent())
	if err != nil {
		v1.HandleError(ctx, statusFor(err), err, nil)
		return
	}
	h.writeAuthCookies(ctx, token)
	v1.HandleSuccess(ctx, resp)
}

func (h *AuthHandler) Logout(ctx *gin.Context) {
	if session := GetSessionFromCtx(ctx); session != nil {
		if err := h.svc.Logout(ctx, session.ID); err != nil {
			h.logger.Warn("revoke session failed: " + err.Error())
		}
	}
	opts := authcookie.OptionsFrom(h.conf, ctx.Request)
	authcookie.ClearSession(ctx.Writer, opts)
	authcookie.ClearCSRF(ctx.Writer, opts)
	v1.HandleSuccess(ctx, nil)
}

// Meta 返回部署能力，前端据此决定是否展示邀请码、验证码和配额提示。
func (h *AuthHandler) Meta(ctx *gin.Context) {
	opts := authcookie.OptionsFrom(h.conf, ctx.Request)
	if authcookie.ReadCSRF(ctx.Request) == "" {
		if token, err := authcookie.NewToken(); err == nil {
			authcookie.WriteCSRF(ctx.Writer, token, opts)
		}
	}
	registration := strings.TrimSpace(h.conf.GetString("security.registration"))
	if registration == "" {
		registration = "open"
	}
	ttlHours := h.conf.GetInt("security.session_ttl_hours")
	if ttlHours <= 0 {
		ttlHours = 168
	}
	env := h.conf.GetString("env")
	if env == "" {
		env = "dev"
	}
	v1.HandleSuccess(ctx, &v1.MetaResponse{
		Version:           "0.1.0",
		Env:               env,
		TermsVersion:      v1.CurrentTermsVersion,
		Registration:      registration,
		InviteRequired:    registration == "invite",
		Captcha:           false,
		EmailVerification: false,
		Features: map[string]bool{
			"sessions":       true,
			"admin_security": true,
		},
		Limits: v1.MetaLimits{
			MaxActiveTasks:  h.conf.GetInt("task.max_active_per_user"),
			DailyTasks:      h.conf.GetInt("task.daily_tasks_per_user"),
			MaxPendingTasks: h.conf.GetInt("task.max_pending_per_user"),
			SessionTTLHours: ttlHours,
		},
	})
}

func (h *AuthHandler) writeAuthCookies(ctx *gin.Context, token string) {
	opts := authcookie.OptionsFrom(h.conf, ctx.Request)
	authcookie.WriteSession(ctx.Writer, token, opts)
	if csrf, err := authcookie.NewToken(); err == nil {
		authcookie.WriteCSRF(ctx.Writer, csrf, opts)
	}
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
	case errors.Is(err, v1.ErrAccountDisabled), errors.Is(err, v1.ErrForbidden), errors.Is(err, v1.ErrCSRFInvalid):
		return http.StatusForbidden
	case errors.Is(err, v1.ErrNotFound), errors.Is(err, v1.ErrTaskNotFound),
		errors.Is(err, v1.ErrCredentialNotFound), errors.Is(err, v1.ErrSessionNotFound):
		return http.StatusNotFound
	case errors.Is(err, v1.ErrTooManyRequests), errors.Is(err, v1.ErrDailyTaskLimit),
		errors.Is(err, v1.ErrPendingTaskLimit), errors.Is(err, v1.ErrTaskLimit),
		errors.Is(err, v1.ErrQueueFull):
		return http.StatusTooManyRequests
	case errors.Is(err, v1.ErrUserAlreadyExists), errors.Is(err, v1.ErrUnsupportedPlatform),
		errors.Is(err, v1.ErrBadRequest), errors.Is(err, v1.ErrCredentialEncrypt),
		errors.Is(err, v1.ErrTaskNotCancelable), errors.Is(err, v1.ErrRegistrationClosed),
		errors.Is(err, v1.ErrInvalidTaskParams), errors.Is(err, v1.ErrInviteInvalid),
		errors.Is(err, v1.ErrTermsRequired):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

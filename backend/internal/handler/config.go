package handler

import (
	"net/http"
	"sort"

	v1 "backend/api/v1"
	"backend/pkg/config"
	"backend/pkg/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ConfigHandler 提供管理员可编辑的安全与配额配置。
// 只允许修改 config.Store 中的白名单字段，密钥和数据库连接永远不会暴露。
type ConfigHandler struct {
	store  *config.Store
	logger *log.Logger
}

func NewConfigHandler(store *config.Store, logger *log.Logger) *ConfigHandler {
	return &ConfigHandler{store: store, logger: logger}
}

func (h *ConfigHandler) Get(ctx *gin.Context) {
	v1.HandleSuccess(ctx, h.store.Snapshot())
}

func (h *ConfigHandler) Update(ctx *gin.Context) {
	var req struct {
		Values map[string]any `json:"values"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	snapshot, err := h.store.Update(req.Values)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	keys := make([]string, 0, len(req.Values))
	for key := range req.Values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	// 只记录键名，不记录邀请码等具体值。
	h.logger.Info("config updated",
		zap.String("user_id", GetUserIdFromCtx(ctx)),
		zap.Strings("keys", keys),
	)
	v1.HandleSuccess(ctx, snapshot)
}

func (h *ConfigHandler) Reload(ctx *gin.Context) {
	snapshot, err := h.store.Reload()
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.logger.Info("config reloaded", zap.String("user_id", GetUserIdFromCtx(ctx)))
	v1.HandleSuccess(ctx, snapshot)
}

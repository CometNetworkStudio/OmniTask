package handler

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"backend/internal/service"
	"backend/internal/ws"
	"backend/pkg/log"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

type WSHandler struct {
	hub      *ws.Hub
	progress *service.ProgressStore
	tasks    *service.TaskService
	conf     *viper.Viper
	logger   *log.Logger
	upgrader websocket.Upgrader
}

func NewWSHandler(hub *ws.Hub, progress *service.ProgressStore, tasks *service.TaskService, conf *viper.Viper, logger *log.Logger) *WSHandler {
	h := &WSHandler{hub: hub, progress: progress, tasks: tasks, conf: conf, logger: logger}
	h.upgrader = websocket.Upgrader{CheckOrigin: h.checkOrigin}
	return h
}

func (h *WSHandler) Handle(c *gin.Context) {
	taskID := c.Query("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}
	// 只有任务所有者能订阅进度，避免知道 task_id 就能窥探他人任务。
	if _, err := h.tasks.Get(c.Request.Context(), GetUserIdFromCtx(c), taskID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(32 << 10)

	ch, cancel := h.hub.Subscribe(taskID)
	defer cancel()

	if snapshot, ok := h.progress.Get(c.Request.Context(), taskID); ok {
		_ = conn.WriteJSON(snapshot)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-done:
			return
		case event := <-ch:
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		case <-ping.C:
			_ = conn.WriteMessage(websocket.PingMessage, nil)
		}
	}
}

// checkOrigin 只允许同源或显式配置的来源。开发环境额外允许 localhost，方便 Vite 代理。
func (h *WSHandler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	allowed := map[string]bool{}
	for _, key := range []string{"security.ws_allowed_origins", "http.cors_origins"} {
		for _, item := range strings.Split(h.conf.GetString(key), ",") {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				allowed[trimmed] = true
			}
		}
	}
	if allowed[origin] {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	host := r.Host
	if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
		host = forwarded
	}
	if strings.EqualFold(u.Host, host) {
		return true
	}
	if loopbackHostname(u.Hostname()) && loopbackRemote(r.RemoteAddr) {
		return true
	}
	if h.conf.GetString("env") != "prod" && loopbackHostname(u.Hostname()) {
		return true
	}
	return false
}

func loopbackHostname(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func loopbackRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	return loopbackHostname(host)
}

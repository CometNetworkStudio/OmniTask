package handler

import (
	"net/http"
	"time"

	"backend/internal/service"
	"backend/internal/ws"
	"backend/pkg/log"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WSHandler struct {
	hub      *ws.Hub
	progress *service.ProgressStore
	logger   *log.Logger
}

func NewWSHandler(hub *ws.Hub, progress *service.ProgressStore, logger *log.Logger) *WSHandler {
	return &WSHandler{hub: hub, progress: progress, logger: logger}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *WSHandler) Handle(c *gin.Context) {
	taskID := c.Query("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

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

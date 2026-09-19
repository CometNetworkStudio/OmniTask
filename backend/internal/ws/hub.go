package ws

import "sync"

type Event map[string]interface{}

// Hub 按 task_id 广播任务进度事件（进程内）。
type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[chan Event]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[string]map[chan Event]struct{})}
}

func (h *Hub) Subscribe(taskID string) (<-chan Event, func()) {
	ch := make(chan Event, 16)
	h.mu.Lock()
	if h.subs[taskID] == nil {
		h.subs[taskID] = make(map[chan Event]struct{})
	}
	h.subs[taskID][ch] = struct{}{}
	h.mu.Unlock()
	cancel := func() {
		h.mu.Lock()
		if set, ok := h.subs[taskID]; ok {
			delete(set, ch)
			if len(set) == 0 {
				delete(h.subs, taskID)
			}
		}
		h.mu.Unlock()
		close(ch)
	}
	return ch, cancel
}

func (h *Hub) Publish(taskID string, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[taskID] {
		select {
		case ch <- event:
		default:
		}
	}
}

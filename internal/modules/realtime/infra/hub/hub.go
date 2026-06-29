// Package hub 是进程内 WebSocket 连接注册表（单机版，支持多端）。
package hub

import (
	"sync"

	"github.com/maguowei/gotobeta/internal/pkg/imrt"
)

// Hub 维护 userID → 多连接 的注册表，线程安全，实现 imrt.Registry。
type Hub struct {
	mu    sync.RWMutex
	conns map[int64]map[imrt.Connection]struct{}
}

// New 创建 Hub。
func New() *Hub {
	return &Hub{conns: make(map[int64]map[imrt.Connection]struct{})}
}

// Register 注册某用户的一条连接（同一用户多端各一条）。
func (h *Hub) Register(userID int64, c imrt.Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.conns[userID]
	if !ok {
		set = make(map[imrt.Connection]struct{})
		h.conns[userID] = set
	}
	set[c] = struct{}{}
}

// Unregister 注销某用户的一条连接；该用户无连接时清理 map 项。
func (h *Hub) Unregister(userID int64, c imrt.Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.conns[userID]
	if !ok {
		return
	}
	delete(set, c)
	if len(set) == 0 {
		delete(h.conns, userID)
	}
}

// Push 向某用户的全部连接投递一帧。
func (h *Hub) Push(userID int64, frame []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.conns[userID] {
		c.Send(frame)
	}
}

// Broadcast 向多个用户的全部连接投递一帧。
func (h *Hub) Broadcast(userIDs []int64, frame []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, uid := range userIDs {
		for c := range h.conns[uid] {
			c.Send(frame)
		}
	}
}

// IsOnline 返回某用户是否有活跃连接。
func (h *Hub) IsOnline(userID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns[userID]) > 0
}

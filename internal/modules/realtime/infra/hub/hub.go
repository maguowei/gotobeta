// Package hub 是进程内 WebSocket 连接注册表（单机版，支持多端）。
package hub

import (
	"sync"

	"github.com/maguowei/gotobeta/internal/pkg/imrt"
)

// Hub 维护 userID → 多连接 的注册表，线程安全，实现 imrt.Registry。
// maxTotal / maxPerUser 为连接数上限（<=0 表示不限），用于过载保护。
type Hub struct {
	mu         sync.RWMutex
	conns      map[int64]map[imrt.Connection]struct{}
	total      int
	maxTotal   int
	maxPerUser int
}

// New 创建 Hub。maxTotal 为全局连接上限，maxPerUser 为单用户连接上限；<=0 表示不限。
func New(maxTotal, maxPerUser int) *Hub {
	return &Hub{
		conns:      make(map[int64]map[imrt.Connection]struct{}),
		maxTotal:   maxTotal,
		maxPerUser: maxPerUser,
	}
}

// Register 注册某用户的一条连接（同一用户多端各一条）。
// 超过全局或单用户上限时拒绝并返回 false，调用方应据此断开连接。
func (h *Hub) Register(userID int64, c imrt.Connection) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.maxTotal > 0 && h.total >= h.maxTotal {
		return false
	}
	set, ok := h.conns[userID]
	if h.maxPerUser > 0 && len(set) >= h.maxPerUser {
		return false
	}
	if !ok {
		set = make(map[imrt.Connection]struct{})
		h.conns[userID] = set
	}
	set[c] = struct{}{}
	h.total++
	return true
}

// Unregister 注销某用户的一条连接；该用户无连接时清理 map 项。
func (h *Hub) Unregister(userID int64, c imrt.Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.conns[userID]
	if !ok {
		return
	}
	if _, exists := set[c]; !exists {
		return
	}
	delete(set, c)
	h.total--
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

// ConnectionCount 返回当前全局连接总数。
func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.total
}

// UserConnectionCount 返回某用户当前的连接数。
func (h *Hub) UserConnectionCount(userID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns[userID])
}

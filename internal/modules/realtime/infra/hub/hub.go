// Package hub 是进程内 WebSocket 连接注册表（单机版，支持多端）。
package hub

import (
	"context"
	"sync"
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/imrt"
)

// drainPollInterval 是优雅关闭时轮询连接是否排空的间隔。
const drainPollInterval = 20 * time.Millisecond

// ConnGauge 是活跃连接数观测端口（由 infra/metrics.Collectors 实现），可为 nil（不埋点）。
type ConnGauge interface {
	SetWSConnections(n float64)
}

// Hub 维护 userID → 多连接 的注册表，线程安全，实现 imrt.Registry。
// maxTotal / maxPerUser 为连接数上限（<=0 表示不限），用于过载保护。
type Hub struct {
	mu         sync.RWMutex
	conns      map[int64]map[imrt.Connection]struct{}
	total      int
	maxTotal   int
	maxPerUser int
	gauge      ConnGauge
}

// SetConnGauge 注入活跃连接数观测端口（组合根装配时调用）。
func (h *Hub) SetConnGauge(g ConnGauge) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.gauge = g
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
	if h.gauge != nil {
		h.gauge.SetWSConnections(float64(h.total))
	}
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
	if h.gauge != nil {
		h.gauge.SetWSConnections(float64(h.total))
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

// GracefulShutdown 主动关闭全部连接（触发 close 帧下发），并轮询等待连接排空，
// 直到 ConnectionCount 归零或 ctx 取消/超时。连接断开后由各自的读循环负责 Unregister。
func (h *Hub) GracefulShutdown(ctx context.Context) error {
	h.mu.RLock()
	conns := make([]imrt.Connection, 0, h.total)
	for _, set := range h.conns {
		for c := range set {
			conns = append(conns, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range conns {
		c.Close()
	}

	ticker := time.NewTicker(drainPollInterval)
	defer ticker.Stop()
	for {
		if h.ConnectionCount() == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

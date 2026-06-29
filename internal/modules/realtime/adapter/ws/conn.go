package ws

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	sendBufferSize = 64
)

// Conn 是一条 WS 连接，实现 hub.Connection。写操作经缓冲 channel 串行化到单一写泵。
type Conn struct {
	userID     int64
	ws         *websocket.Conn
	send       chan []byte
	closed     chan struct{}
	onOverflow func()
}

// newConn 创建连接。bufSize 为写缓冲容量；onOverflow 在缓冲溢出主动断连时回调（可为 nil，供指标计数）。
func newConn(userID int64, wsConn *websocket.Conn, bufSize int, onOverflow func()) *Conn {
	if bufSize <= 0 {
		bufSize = sendBufferSize
	}
	return &Conn{
		userID:     userID,
		ws:         wsConn,
		send:       make(chan []byte, bufSize),
		closed:     make(chan struct{}),
		onOverflow: onOverflow,
	}
}

// UserID 返回连接绑定的用户。
func (c *Conn) UserID() int64 { return c.userID }

// Send 非阻塞投递一帧。连接已关闭时丢弃；写缓冲已满时不静默丢弃，而是触发溢出钩子
// 并主动断连——客户端重连后按 last_seq 走 HTTP 增量拉取补偿，避免消息空洞。
func (c *Conn) Send(frame []byte) {
	select {
	case <-c.closed:
	case c.send <- frame:
	default:
		if c.onOverflow != nil {
			c.onOverflow()
		}
		c.Close()
	}
}

// writePump 串行写帧并定期发送 ping 维持心跳。
func (c *Conn) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.ws.Close()
	}()
	for {
		select {
		case frame, ok := <-c.send:
			_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.ws.WriteMessage(websocket.TextMessage, frame); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.closed:
			// 主动关闭（含优雅停机）时给客户端发送规范 close 帧，便于其立即重连补拉。
			_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			_ = c.ws.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutting down"))
			return
		}
	}
}

// Close 关闭连接（幂等）。
func (c *Conn) Close() {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
}

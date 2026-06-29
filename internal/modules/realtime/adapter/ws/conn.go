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
	userID int64
	ws     *websocket.Conn
	send   chan []byte
	closed chan struct{}
}

func newConn(userID int64, wsConn *websocket.Conn) *Conn {
	return &Conn{
		userID: userID,
		ws:     wsConn,
		send:   make(chan []byte, sendBufferSize),
		closed: make(chan struct{}),
	}
}

// UserID 返回连接绑定的用户。
func (c *Conn) UserID() int64 { return c.userID }

// Send 非阻塞投递一帧；写队列已满或连接已关闭时丢弃（推送尽力而为，由拉取补偿）。
func (c *Conn) Send(frame []byte) {
	select {
	case <-c.closed:
	case c.send <- frame:
	default:
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
			return
		}
	}
}

// close 关闭连接（幂等）。
func (c *Conn) close() {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
}

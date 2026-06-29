package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/maguowei/gotobeta/internal/modules/realtime/adapter/ws"
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/hub"
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/ticket"
)

func newGatewayServer(t *testing.T) (*httptest.Server, *ticket.Store, *hub.Hub) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tickets := ticket.NewStore(nil, time.Minute)
	connHub := hub.New()
	gw := ws.NewGateway(tickets, connHub, nil, slog.Default())
	engine := gin.New()
	engine.GET("/ws", gw.Handle)
	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)
	return srv, tickets, connHub
}

func wsURL(base, token string) string {
	return "ws" + strings.TrimPrefix(base, "http") + "/ws?ticket=" + token
}

func TestGatewayHandshakeAndPing(t *testing.T) {
	srv, tickets, _ := newGatewayServer(t)
	token, err := tickets.Issue(context.Background(), 42)
	if err != nil {
		t.Fatalf("签发 ticket 失败: %v", err)
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, token), nil)
	if err != nil {
		t.Fatalf("握手失败: %v", err)
	}
	defer conn.Close()

	var f ws.Frame
	if err := readFrame(conn, &f); err != nil {
		t.Fatalf("读取 auth_ok 失败: %v", err)
	}
	if f.T != ws.TypeAuthOK || f.UID != 42 {
		t.Fatalf("应为 auth_ok(uid=42), got %+v", f)
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"t":"ping"}`)); err != nil {
		t.Fatalf("写 ping 失败: %v", err)
	}
	if err := readFrame(conn, &f); err != nil {
		t.Fatalf("读取 pong 失败: %v", err)
	}
	if f.T != ws.TypePong {
		t.Fatalf("应为 pong, got %+v", f)
	}
}

func TestGatewayRejectsInvalidTicket(t *testing.T) {
	srv, _, _ := newGatewayServer(t)
	_, resp, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, "bogus"), nil)
	if err == nil {
		t.Fatal("非法 ticket 应握手失败")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("应返回 401, got %v", resp)
	}
}

func TestGatewaySignalDelivery(t *testing.T) {
	srv, tickets, connHub := newGatewayServer(t)
	token, _ := tickets.Issue(context.Background(), 7)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, token), nil)
	if err != nil {
		t.Fatalf("握手失败: %v", err)
	}
	defer conn.Close()
	var skip ws.Frame
	if err := readFrame(conn, &skip); err != nil {
		t.Fatalf("读取 auth_ok 失败: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for !connHub.IsOnline(7) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	connHub.Push(7, ws.SignalFrame(100, 9))

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var f ws.Frame
	if err := readFrame(conn, &f); err != nil {
		t.Fatalf("读取 signal 失败: %v", err)
	}
	if f.T != ws.TypeSignal || f.CID != 100 || f.Seq != 9 {
		t.Fatalf("应为 signal(cid=100,seq=9), got %+v", f)
	}
}

func readFrame(conn *websocket.Conn, f *ws.Frame) error {
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, f)
}

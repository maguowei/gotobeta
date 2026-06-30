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
	connHub := hub.New(0, 0)
	gw := ws.NewGateway(tickets, connHub, nil, nil, slog.Default(), ws.GatewayConfig{})
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
	defer func() { _ = conn.Close() }()

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
	defer func() { _ = conn.Close() }()
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

func TestGatewayHandshakeCarriesProtocolVersion(t *testing.T) {
	// 缺省（不带 v）的老客户端应按 v1 放行，且 auth_ok 回带服务端协议版本。
	srv, tickets, _ := newGatewayServer(t)
	token, _ := tickets.Issue(context.Background(), 1)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, token), nil)
	if err != nil {
		t.Fatalf("缺省版本握手应成功: %v", err)
	}
	defer func() { _ = conn.Close() }()

	var f ws.Frame
	if err := readFrame(conn, &f); err != nil {
		t.Fatalf("读取 auth_ok 失败: %v", err)
	}
	if f.T != ws.TypeAuthOK || f.PV != ws.CurrentProtocolVersion {
		t.Fatalf("auth_ok 应回带 pv=%d, got %+v", ws.CurrentProtocolVersion, f)
	}
}

func TestGatewayAcceptsCompatibleVersion(t *testing.T) {
	srv, tickets, _ := newGatewayServer(t)
	token, _ := tickets.Issue(context.Background(), 2)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, token)+"&v=1", nil)
	if err != nil {
		t.Fatalf("兼容版本握手应成功: %v", err)
	}
	defer func() { _ = conn.Close() }()

	var f ws.Frame
	if err := readFrame(conn, &f); err != nil {
		t.Fatalf("读取 auth_ok 失败: %v", err)
	}
	if f.T != ws.TypeAuthOK || f.PV != ws.CurrentProtocolVersion {
		t.Fatalf("应为 auth_ok(pv=%d), got %+v", ws.CurrentProtocolVersion, f)
	}
}

func TestGatewayRejectsUnsupportedVersion(t *testing.T) {
	// 不兼容版本在升级后以应用区间 close 码（4000）拒绝，连接不进入读写泵。
	srv, tickets, connHub := newGatewayServer(t)
	token, _ := tickets.Issue(context.Background(), 3)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, token)+"&v=999", nil)
	if err != nil {
		t.Fatalf("升级阶段不应失败（版本不兼容在升级后用 close 帧表达）: %v", err)
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	closeErr, ok := err.(*websocket.CloseError)
	if !ok || closeErr.Code != 4000 {
		t.Fatalf("应收到 close 码 4000, got err=%v", err)
	}
	if connHub.IsOnline(3) {
		t.Fatal("版本不兼容的连接不应注册到 Hub")
	}
}

func readFrame(conn *websocket.Conn, f *ws.Frame) error {
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, f)
}

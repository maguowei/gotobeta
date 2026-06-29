package hub

import (
	"context"
	"sync"
	"testing"
	"time"
)

type fakeConn struct {
	mu     sync.Mutex
	frames [][]byte
	closed bool
}

func (c *fakeConn) Send(frame []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.frames = append(c.frames, frame)
}

func (c *fakeConn) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
}

func (c *fakeConn) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *fakeConn) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.frames)
}

func TestPushToAllConnectionsOfUser(t *testing.T) {
	t.Parallel()
	h := New(0, 0)
	c1, c2 := &fakeConn{}, &fakeConn{}
	h.Register(1, c1)
	h.Register(1, c2)
	h.Push(1, []byte("x"))
	if c1.count() != 1 || c2.count() != 1 {
		t.Fatalf("两端都应收到, got %d %d", c1.count(), c2.count())
	}
}

func TestUnregisterCleansUp(t *testing.T) {
	t.Parallel()
	h := New(0, 0)
	c := &fakeConn{}
	h.Register(1, c)
	if !h.IsOnline(1) {
		t.Fatal("注册后应在线")
	}
	h.Unregister(1, c)
	if h.IsOnline(1) {
		t.Fatal("注销后应离线")
	}
	// 注销后 Push 不 panic 也不投递。
	h.Push(1, []byte("y"))
	if c.count() != 0 {
		t.Fatalf("注销后不应收到, got %d", c.count())
	}
}

func TestBroadcast(t *testing.T) {
	t.Parallel()
	h := New(0, 0)
	c1, c2, c3 := &fakeConn{}, &fakeConn{}, &fakeConn{}
	h.Register(1, c1)
	h.Register(2, c2)
	h.Register(3, c3)
	h.Broadcast([]int64{1, 2}, []byte("z"))
	if c1.count() != 1 || c2.count() != 1 {
		t.Fatalf("目标用户应收到")
	}
	if c3.count() != 0 {
		t.Fatal("非目标用户不应收到")
	}
}

func TestRegisterRejectsOverPerUserLimit(t *testing.T) {
	t.Parallel()
	h := New(0, 1) // 每用户最多 1 条连接
	c1, c2 := &fakeConn{}, &fakeConn{}
	if !h.Register(1, c1) {
		t.Fatal("首条连接应接纳")
	}
	if h.Register(1, c2) {
		t.Fatal("超过单用户上限应拒绝")
	}
	if got := h.UserConnectionCount(1); got != 1 {
		t.Fatalf("单用户连接数 = %d, want 1", got)
	}
}

func TestRegisterRejectsOverTotalLimit(t *testing.T) {
	t.Parallel()
	h := New(1, 0) // 全局最多 1 条连接
	if !h.Register(1, &fakeConn{}) {
		t.Fatal("首条连接应接纳")
	}
	if h.Register(2, &fakeConn{}) {
		t.Fatal("超过全局上限应拒绝")
	}
	if got := h.ConnectionCount(); got != 1 {
		t.Fatalf("全局连接数 = %d, want 1", got)
	}
}

// drainConn 在 Close 时把自己从 Hub 注销，模拟真实连接断开后 readPump 触发的 Unregister。
type drainConn struct {
	fakeConn
	h   *Hub
	uid int64
}

func (d *drainConn) Close() {
	d.fakeConn.Close()
	d.h.Unregister(d.uid, d)
}

func TestGracefulShutdownClosesAndDrains(t *testing.T) {
	t.Parallel()
	h := New(0, 0)
	d1 := &drainConn{h: h, uid: 1}
	d2 := &drainConn{h: h, uid: 2}
	h.Register(1, d1)
	h.Register(2, d2)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := h.GracefulShutdown(ctx); err != nil {
		t.Fatalf("应正常排空，得 %v", err)
	}
	if !d1.isClosed() || !d2.isClosed() {
		t.Fatal("所有连接应被 Close")
	}
	if got := h.ConnectionCount(); got != 0 {
		t.Fatalf("排空后连接数应为 0，得 %d", got)
	}
}

func TestGracefulShutdownTimesOutWhenStuck(t *testing.T) {
	t.Parallel()
	h := New(0, 0)
	h.Register(1, &fakeConn{}) // fakeConn.Close 不注销，连接无法排空
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := h.GracefulShutdown(ctx); err == nil {
		t.Fatal("无法排空时应返回超时错误")
	}
}

func TestRegisterRejectedAfterShutdown(t *testing.T) {
	t.Parallel()
	h := New(0, 0)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 空 Hub 立即排空完成，但 shutting 标记保持，后续 Register 必须被拒绝。
	if err := h.GracefulShutdown(ctx); err != nil {
		t.Fatalf("空 Hub 应立即排空，得 %v", err)
	}
	if h.Register(1, &fakeConn{}) {
		t.Fatal("优雅关闭后不应再接受新连接")
	}
}

type countGauge struct{ last float64 }

func (g *countGauge) SetWSConnections(n float64) { g.last = n }

func TestConnGaugeUpdatedOnRegisterUnregister(t *testing.T) {
	t.Parallel()
	h := New(0, 0)
	g := &countGauge{}
	h.SetConnGauge(g)
	c := &fakeConn{}
	h.Register(1, c)
	if g.last != 1 {
		t.Fatalf("注册后 gauge 应为 1，得 %v", g.last)
	}
	h.Unregister(1, c)
	if g.last != 0 {
		t.Fatalf("注销后 gauge 应为 0，得 %v", g.last)
	}
}

func TestConnectionCountTracksUnregister(t *testing.T) {
	t.Parallel()
	h := New(0, 0) // 无上限
	c1, c2 := &fakeConn{}, &fakeConn{}
	h.Register(1, c1)
	h.Register(1, c2)
	if got := h.ConnectionCount(); got != 2 {
		t.Fatalf("注册后全局连接数 = %d, want 2", got)
	}
	h.Unregister(1, c1)
	if got := h.ConnectionCount(); got != 1 {
		t.Fatalf("注销一条后 = %d, want 1", got)
	}
	if got := h.UserConnectionCount(1); got != 1 {
		t.Fatalf("注销一条后单用户 = %d, want 1", got)
	}
}

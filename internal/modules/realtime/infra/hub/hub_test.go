package hub

import (
	"sync"
	"testing"
)

type fakeConn struct {
	mu     sync.Mutex
	frames [][]byte
}

func (c *fakeConn) Send(frame []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.frames = append(c.frames, frame)
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

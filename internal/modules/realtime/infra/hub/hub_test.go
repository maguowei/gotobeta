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
	h := New()
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
	h := New()
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
	h := New()
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

package ws

import (
	"sync/atomic"
	"testing"
)

func TestSendOverflowTriggersCloseAndHook(t *testing.T) {
	var hookCalls int32
	// bufSize=1：写满后再写触发溢出。wsConn 为 nil，因 Send 只操作 channel，不触碰底层连接。
	c := newConn(1, nil, 1, func() { atomic.AddInt32(&hookCalls, 1) }, wsTimeouts{})

	c.Send([]byte("a")) // 占满缓冲（cap=1）
	c.Send([]byte("b")) // 缓冲已满 → 溢出 → 钩子 + 断连

	select {
	case <-c.closed:
		// 期望：已关闭
	default:
		t.Fatal("缓冲溢出后连接应被主动关闭")
	}
	if got := atomic.LoadInt32(&hookCalls); got != 1 {
		t.Fatalf("onOverflow 应被调用一次，得 %d", got)
	}
}

func TestSendAfterCloseIsNoop(t *testing.T) {
	c := newConn(1, nil, 1, nil, wsTimeouts{})
	c.Close()
	// 已关闭后发送不应 panic，也不触发钩子（nil hook）。
	c.Send([]byte("x"))
}

func TestSendWithinBufferDoesNotOverflow(t *testing.T) {
	var hookCalls int32
	c := newConn(1, nil, 4, func() { atomic.AddInt32(&hookCalls, 1) }, wsTimeouts{})
	c.Send([]byte("a"))
	c.Send([]byte("b"))
	select {
	case <-c.closed:
		t.Fatal("未溢出不应关闭")
	default:
	}
	if got := atomic.LoadInt32(&hookCalls); got != 0 {
		t.Fatalf("未溢出不应触发钩子，得 %d", got)
	}
}

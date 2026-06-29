package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/modules/realtime/adapter/ws"
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/hub"
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/presence"
	"github.com/maguowei/gotobeta/internal/pkg/imevent"
)

// errMembers 让成员查询返回错误，覆盖各调用方的 lookup 失败分支。
type errMembers struct{ err error }

func (m errMembers) ConversationUserIDs(context.Context, int64) ([]int64, error) {
	return nil, m.err
}

func (m errMembers) UserConversationPeers(context.Context, int64) ([]int64, error) {
	return nil, m.err
}

// errReader 让 ReportRead 返回错误，覆盖 Read 的失败分支。
type errReader struct{ err error }

func (r errReader) ReportRead(context.Context, int64, int64, int64) error { return r.err }

func TestDispatcherIgnoresUnrelatedEvent(t *testing.T) {
	d := NewDispatcher(hub.New(0, 0), stubMembers{}, slog.Default(), nil)
	// 非 MessageCreatedEvent 应被忽略（返回 nil，不触发查询）。
	if err := d.OnMessageCreated(context.Background(), imevent.ReadUpdatedEvent{}); err != nil {
		t.Fatalf("无关事件应被忽略, got %v", err)
	}
	if err := d.OnReadUpdated(context.Background(), imevent.MessageCreatedEvent{}); err != nil {
		t.Fatalf("无关事件应被忽略, got %v", err)
	}
}

func TestDispatcherOnMessageCreatedLookupError(t *testing.T) {
	boom := errors.New("lookup boom")
	d := NewDispatcher(hub.New(0, 0), errMembers{err: boom}, slog.Default(), nil)
	evt := imevent.NewMessageCreatedEvent(7, 100, 8001, 5, 1, 9, 1, time.Now())
	if err := d.OnMessageCreated(context.Background(), evt); !errors.Is(err, boom) {
		t.Fatalf("查询失败应透传, got %v", err)
	}
}

func TestDispatcherOnReadUpdatedPushesReadFrame(t *testing.T) {
	h := hub.New(0, 0)
	member := &recvConn{}
	h.Register(1, member)

	d := NewDispatcher(h, stubMembers{ids: []int64{1}}, slog.Default(), nil)
	evt := imevent.NewReadUpdatedEvent(7, 100, 1, 12, time.Now())
	if err := d.OnReadUpdated(context.Background(), evt); err != nil {
		t.Fatalf("分发失败: %v", err)
	}

	if len(member.frames) != 1 {
		t.Fatalf("成员应收到 1 帧, got %d", len(member.frames))
	}
	var f ws.Frame
	if err := json.Unmarshal(member.frames[0], &f); err != nil {
		t.Fatalf("帧解析失败: %v", err)
	}
	if f.T != ws.TypeRead || f.CID != 100 || f.UID != 1 || f.ReadSeq != 12 {
		t.Fatalf("read 帧错误: %+v", f)
	}
}

func TestDispatcherOnReadUpdatedLookupError(t *testing.T) {
	boom := errors.New("lookup boom")
	d := NewDispatcher(hub.New(0, 0), errMembers{err: boom}, slog.Default(), nil)
	evt := imevent.NewReadUpdatedEvent(7, 100, 1, 12, time.Now())
	if err := d.OnReadUpdated(context.Background(), evt); !errors.Is(err, boom) {
		t.Fatalf("查询失败应透传, got %v", err)
	}
}

func TestTypingLookupErrorIsSwallowed(t *testing.T) {
	h := hub.New(0, 0)
	peer := &recvConn{}
	h.Register(2, peer)
	e := NewEphemeral(h, errMembers{err: errors.New("boom")}, &captureReader{}, slog.Default())

	// 查询失败时静默返回，不广播任何帧。
	e.Typing(context.Background(), 1, 100)
	if len(peer.frames) != 0 {
		t.Fatalf("查询失败不应广播, got %d", len(peer.frames))
	}
}

func TestReadErrorIsSwallowed(t *testing.T) {
	e := NewEphemeral(hub.New(0, 0), stubMembers{}, errReader{err: errors.New("boom")}, slog.Default())
	// ReportRead 失败仅记日志，不 panic。
	e.Read(context.Background(), 9, 100, 12)
}

func TestPresenceOnDisconnectMarksOfflineWhenNoConn(t *testing.T) {
	h := hub.New(0, 0)
	peer := &recvConn{}
	h.Register(2, peer)
	store := presence.NewStore(nil, 0)
	p := NewPresence(h, store, stubMembers{peers: []int64{2}}, slog.Default())

	// 用户 1 无任何连接 → 视为离线并广播 offline。
	p.OnDisconnect(context.Background(), 1)

	if len(peer.frames) != 1 {
		t.Fatalf("同伴应收到 offline, got %d", len(peer.frames))
	}
	var f ws.Frame
	_ = json.Unmarshal(peer.frames[0], &f)
	if f.T != ws.TypePresence || f.UID != 1 || f.Online == nil || *f.Online {
		t.Fatalf("offline 帧错误: %+v", f)
	}
}

func TestPresenceOnDisconnectKeepsOnlineWhenOtherConn(t *testing.T) {
	h := hub.New(0, 0)
	peer := &recvConn{}
	h.Register(2, peer)
	h.Register(1, &recvConn{}) // 用户 1 仍有其他端在线

	p := NewPresence(h, presence.NewStore(nil, 0), stubMembers{peers: []int64{2}}, slog.Default())
	p.OnDisconnect(context.Background(), 1)

	// 多端仍在线时不广播 offline。
	if len(peer.frames) != 0 {
		t.Fatalf("仍有连接时不应广播 offline, got %d", len(peer.frames))
	}
}

// errKV 让 presence.Store 的 MarkOnline/MarkOffline 返回错误，覆盖标记失败分支。
type errKV struct{ err error }

func (k errKV) Set(context.Context, string, string, time.Duration) error { return k.err }
func (k errKV) Del(context.Context, string) error                        { return k.err }

func TestPresenceOnConnectMarkOnlineError(t *testing.T) {
	h := hub.New(0, 0)
	peer := &recvConn{}
	h.Register(2, peer)
	store := presence.NewStore(errKV{err: errors.New("boom")}, time.Minute)
	p := NewPresence(h, store, stubMembers{peers: []int64{2}}, slog.Default())

	// 标记上线失败仅记日志，仍广播 online。
	p.OnConnect(context.Background(), 1)
	if len(peer.frames) != 1 {
		t.Fatalf("标记失败仍应广播 online, got %d", len(peer.frames))
	}
}

func TestPresenceOnDisconnectMarkOfflineError(t *testing.T) {
	h := hub.New(0, 0)
	peer := &recvConn{}
	h.Register(2, peer)
	store := presence.NewStore(errKV{err: errors.New("boom")}, time.Minute)
	p := NewPresence(h, store, stubMembers{peers: []int64{2}}, slog.Default())

	// 用户 1 无连接 → 标记离线失败仅记日志，仍广播 offline。
	p.OnDisconnect(context.Background(), 1)
	if len(peer.frames) != 1 {
		t.Fatalf("标记失败仍应广播 offline, got %d", len(peer.frames))
	}
}

func TestPresenceBroadcastLookupErrorIsSwallowed(t *testing.T) {
	h := hub.New(0, 0)
	peer := &recvConn{}
	h.Register(2, peer)
	p := NewPresence(h, presence.NewStore(nil, 0), errMembers{err: errors.New("boom")}, slog.Default())

	// peers 查询失败时静默返回，不广播。
	p.OnConnect(context.Background(), 1)
	if len(peer.frames) != 0 {
		t.Fatalf("查询失败不应广播, got %d", len(peer.frames))
	}
}

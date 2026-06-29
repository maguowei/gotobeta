package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/maguowei/gotobeta/internal/modules/realtime/adapter/ws"
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/hub"
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/presence"
)

type captureReader struct {
	called bool
	convID int64
	userID int64
	seq    int64
}

func (r *captureReader) ReportRead(_ context.Context, conversationID, userID, readSeq int64) error {
	r.called = true
	r.convID, r.userID, r.seq = conversationID, userID, readSeq
	return nil
}

func TestTypingBroadcastExcludesSelf(t *testing.T) {
	h := hub.New()
	self, peer := &recvConn{}, &recvConn{}
	h.Register(1, self)
	h.Register(2, peer)
	e := NewEphemeral(h, stubMembers{ids: []int64{1, 2}}, &captureReader{}, slog.Default())

	e.Typing(context.Background(), 1, 100)

	if len(self.frames) != 0 {
		t.Fatal("typing 不应回送自己")
	}
	if len(peer.frames) != 1 {
		t.Fatalf("同伴应收到 typing, got %d", len(peer.frames))
	}
	var f ws.Frame
	_ = json.Unmarshal(peer.frames[0], &f)
	if f.T != ws.TypeTyping || f.CID != 100 || f.UID != 1 {
		t.Fatalf("typing 帧错误: %+v", f)
	}
}

func TestReadEphemeralForwards(t *testing.T) {
	reader := &captureReader{}
	e := NewEphemeral(hub.New(), stubMembers{}, reader, slog.Default())
	e.Read(context.Background(), 9, 100, 12)
	if !reader.called || reader.convID != 100 || reader.userID != 9 || reader.seq != 12 {
		t.Fatalf("read 应回流到 reader: %+v", reader)
	}
}

func TestPresenceBroadcastsToPeers(t *testing.T) {
	h := hub.New()
	peer := &recvConn{}
	h.Register(2, peer)
	p := NewPresence(h, presence.NewStore(nil, 0), stubMembers{peers: []int64{2}}, slog.Default())

	p.OnConnect(context.Background(), 1)

	if len(peer.frames) != 1 {
		t.Fatalf("同伴应收到 presence, got %d", len(peer.frames))
	}
	var f ws.Frame
	_ = json.Unmarshal(peer.frames[0], &f)
	if f.T != ws.TypePresence || f.UID != 1 || f.Online == nil || !*f.Online {
		t.Fatalf("presence 帧错误: %+v", f)
	}
}

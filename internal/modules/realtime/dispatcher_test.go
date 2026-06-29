package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/hub"
	"github.com/maguowei/gotobeta/internal/pkg/imevent"
)

type stubMembers struct {
	ids   []int64
	peers []int64
}

func (s stubMembers) ConversationUserIDs(context.Context, int64) ([]int64, error) {
	return s.ids, nil
}

func (s stubMembers) UserConversationPeers(context.Context, int64) ([]int64, error) {
	return s.peers, nil
}

type recvConn struct{ frames [][]byte }

func (c *recvConn) Send(frame []byte) { c.frames = append(c.frames, frame) }
func (c *recvConn) Close()            {}

func TestDispatcherPushesSignalToOnlineMembers(t *testing.T) {
	h := hub.New(0, 0)
	online := &recvConn{}
	h.Register(1, online) // 用户 1 在线
	// 用户 2 离线（不注册）

	d := NewDispatcher(h, stubMembers{ids: []int64{1, 2}}, slog.Default())
	evt := imevent.NewMessageCreatedEvent(7, 100, 8001, 5, 1, 9, 1, time.Now())
	if err := d.OnMessageCreated(context.Background(), evt); err != nil {
		t.Fatalf("分发失败: %v", err)
	}

	if len(online.frames) != 1 {
		t.Fatalf("在线成员应收到 1 帧, got %d", len(online.frames))
	}
	var f map[string]any
	if err := json.Unmarshal(online.frames[0], &f); err != nil {
		t.Fatalf("帧解析失败: %v", err)
	}
	if f["t"] != "signal" {
		t.Fatalf("应为 signal 帧, got %v", f["t"])
	}
	if int64(f["cid"].(float64)) != 100 || int64(f["seq"].(float64)) != 5 {
		t.Fatalf("signal 内容错误: %v", f)
	}
}

package ws

import (
	"encoding/json"
	"testing"
)

func TestFrameConstructors(t *testing.T) {
	check := func(raw []byte, wantType string) Frame {
		var f Frame
		if err := json.Unmarshal(raw, &f); err != nil {
			t.Fatalf("解析失败: %v", err)
		}
		if f.T != wantType {
			t.Fatalf("帧类型错误: got %q want %q", f.T, wantType)
		}
		return f
	}
	if f := check(SignalFrame(100, 9), TypeSignal); f.CID != 100 || f.Seq != 9 {
		t.Fatalf("signal 内容错误: %+v", f)
	}
	if f := check(TypingFrame(100, 7), TypeTyping); f.CID != 100 || f.UID != 7 {
		t.Fatalf("typing 内容错误: %+v", f)
	}
	if f := check(ReadFrame(100, 7, 12), TypeRead); f.ReadSeq != 12 {
		t.Fatalf("read 内容错误: %+v", f)
	}
	if f := check(PresenceFrame(7, true), TypePresence); f.Online == nil || !*f.Online {
		t.Fatalf("presence 内容错误: %+v", f)
	}
	check(pongFrame(), TypePong)
}

func TestDecodeFrame(t *testing.T) {
	f, err := decodeFrame([]byte(`{"t":"ping"}`))
	if err != nil || f.T != TypePing {
		t.Fatalf("decode 错误: %+v %v", f, err)
	}
	if _, err := decodeFrame([]byte(`{bad`)); err == nil {
		t.Fatal("非法 JSON 应报错")
	}
}

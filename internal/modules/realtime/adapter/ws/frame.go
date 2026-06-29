package ws

import "encoding/json"

// 帧类型常量（spec 6，第一期 JSON）。
const (
	// 上行。
	TypeAuth   = "auth"
	TypePing   = "ping"
	TypeTyping = "typing"
	TypeRead   = "read"
	// 下行。
	TypeAuthOK   = "auth_ok"
	TypeAuthErr  = "auth_err"
	TypePong     = "pong"
	TypeSignal   = "signal"
	TypePresence = "presence"
)

// Frame 是 WS 协议帧，上下行复用同一结构（字段按帧类型取舍）。
type Frame struct {
	T       string `json:"t"`
	CID     int64  `json:"cid,omitempty"`
	Seq     int64  `json:"seq,omitempty"`
	UID     int64  `json:"uid,omitempty"`
	ReadSeq int64  `json:"read_seq,omitempty"`
	Online  *bool  `json:"online,omitempty"`
	Ticket  string `json:"ticket,omitempty"`
}

// decodeFrame 解析上行帧。
func decodeFrame(raw []byte) (Frame, error) {
	var f Frame
	err := json.Unmarshal(raw, &f)
	return f, err
}

func mustEncode(f Frame) []byte {
	buf, _ := json.Marshal(f)
	return buf
}

// SignalFrame 构造「有新消息信号」下行帧。
func SignalFrame(cid, seq int64) []byte {
	return mustEncode(Frame{T: TypeSignal, CID: cid, Seq: seq})
}

// TypingFrame 构造 typing 广播下行帧。
func TypingFrame(cid, uid int64) []byte {
	return mustEncode(Frame{T: TypeTyping, CID: cid, UID: uid})
}

// ReadFrame 构造已读水位更新下行帧。
func ReadFrame(cid, uid, readSeq int64) []byte {
	return mustEncode(Frame{T: TypeRead, CID: cid, UID: uid, ReadSeq: readSeq})
}

// PresenceFrame 构造在线状态变更下行帧。
func PresenceFrame(uid int64, online bool) []byte {
	return mustEncode(Frame{T: TypePresence, UID: uid, Online: &online})
}

func pongFrame() []byte { return mustEncode(Frame{T: TypePong}) }

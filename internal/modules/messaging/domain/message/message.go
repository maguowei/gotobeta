// Package message 是消息聚合：会话 Timeline 的类型化条目。
//
// 聚合边界 = 包边界；消息按 seq 在会话内连续递增，撤回/系统提示同为带 seq 的条目。
package message

import (
	"time"
	"unicode/utf8"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// SenderType 表示发送者主体类型。
type SenderType int8

const (
	// SenderUser 普通用户。
	SenderUser SenderType = 1
	// SenderBot 机器人（AI 一等公民）。
	SenderBot SenderType = 2
	// SenderSystem 系统条目（撤回/成员变更提示）。
	SenderSystem SenderType = 3
)

// ContentType 表示消息内容类型。
type ContentType int8

const (
	// ContentText 文本。
	ContentText ContentType = 1
	// ContentImage 图片。
	ContentImage ContentType = 2
	// ContentFile 文件。
	ContentFile ContentType = 3
	// ContentVoice 语音。
	ContentVoice ContentType = 4
	// ContentRecall 撤回控制条目。
	ContentRecall ContentType = 10
	// ContentSystem 系统提示。
	ContentSystem ContentType = 11
	// ContentCard 卡片。
	ContentCard ContentType = 20
)

// Status 表示消息状态。
type Status int8

const (
	// StatusNormal 正常。
	StatusNormal Status = 1
	// StatusRecalled 已撤回。
	StatusRecalled Status = 2
	// StatusDeleted 已删除。
	StatusDeleted Status = 3
)

// Message 是消息聚合根。
type Message struct {
	id             int64
	conversationID int64
	seq            int64
	senderType     SenderType
	senderID       int64
	clientMsgID    *string
	contentType    ContentType
	content        map[string]any
	replyToMsgID   int64
	status         Status
	serverTime     time.Time
	metadata       map[string]any
	createdAt      time.Time
	updatedAt      time.Time
}

func (m *Message) ID() int64                { return m.id }
func (m *Message) ConversationID() int64    { return m.conversationID }
func (m *Message) Seq() int64               { return m.seq }
func (m *Message) SenderType() SenderType   { return m.senderType }
func (m *Message) SenderID() int64          { return m.senderID }
func (m *Message) ClientMsgID() *string     { return m.clientMsgID }
func (m *Message) ContentType() ContentType { return m.contentType }
func (m *Message) Content() map[string]any  { return m.content }
func (m *Message) ReplyToMsgID() int64      { return m.replyToMsgID }
func (m *Message) Status() Status           { return m.status }
func (m *Message) ServerTime() time.Time    { return m.serverTime }
func (m *Message) Metadata() map[string]any { return m.metadata }
func (m *Message) CreatedAt() time.Time     { return m.createdAt }
func (m *Message) UpdatedAt() time.Time     { return m.updatedAt }

// New 创建一条用户/机器人消息。seq 由应用层在事务内分配后传入。
func New(id, conversationID, seq int64, senderType SenderType, senderID int64, clientMsgID string, contentType ContentType, content map[string]any, replyToMsgID int64) (*Message, error) {
	if senderType != SenderUser && senderType != SenderBot {
		return nil, apperr.InvalidParam("发送者类型非法")
	}
	if !isSendableContentType(contentType) {
		return nil, apperr.InvalidParam("消息内容类型非法")
	}
	if len(content) == 0 {
		return nil, apperr.InvalidParam("消息内容不能为空")
	}
	if contentType == ContentText {
		if text, _ := content["text"].(string); text == "" {
			return nil, apperr.InvalidParam("文本消息内容不能为空")
		}
	}
	now := time.Now()
	var clientID *string
	if clientMsgID != "" {
		clientID = &clientMsgID
	}
	return &Message{
		id: id, conversationID: conversationID, seq: seq,
		senderType: senderType, senderID: senderID, clientMsgID: clientID,
		contentType: contentType, content: content, replyToMsgID: replyToMsgID,
		status: StatusNormal, serverTime: now, metadata: map[string]any{},
		createdAt: now, updatedAt: now,
	}, nil
}

// NewSystem 创建系统控制条目（撤回/成员变更提示），同样占用一个 seq 进 timeline。
func NewSystem(id, conversationID, seq int64, contentType ContentType, content map[string]any) *Message {
	now := time.Now()
	if content == nil {
		content = map[string]any{}
	}
	return &Message{
		id: id, conversationID: conversationID, seq: seq,
		senderType: SenderSystem, senderID: 0, contentType: contentType, content: content,
		status: StatusNormal, serverTime: now, metadata: map[string]any{},
		createdAt: now, updatedAt: now,
	}
}

func isSendableContentType(ct ContentType) bool {
	switch ct {
	case ContentText, ContentImage, ContentFile, ContentVoice, ContentCard:
		return true
	default:
		return false
	}
}

// Recall 在撤回窗口内撤回消息；超窗或状态非正常返回错误。
func (m *Message) Recall(now time.Time, window time.Duration) error {
	if m.status != StatusNormal {
		return ErrNotRecallable
	}
	if now.Sub(m.serverTime) > window {
		return ErrRecallWindowExpired
	}
	m.status = StatusRecalled
	m.updatedAt = now
	return nil
}

// Digest 生成会话列表用的末条消息摘要。
func (m *Message) Digest() string {
	if m.status == StatusRecalled {
		return "撤回了一条消息"
	}
	switch m.contentType {
	case ContentText:
		text, _ := m.content["text"].(string)
		return truncate(text, 60)
	case ContentImage:
		return "[图片]"
	case ContentFile:
		return "[文件]"
	case ContentVoice:
		return "[语音]"
	case ContentCard:
		return "[卡片]"
	default:
		return ""
	}
}

func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}

// UnmarshalFromDB 从数据库重建消息聚合。
func UnmarshalFromDB(id, conversationID, seq int64, senderType SenderType, senderID int64, clientMsgID *string, contentType ContentType, content map[string]any, replyToMsgID int64, status Status, serverTime time.Time, metadata map[string]any, createdAt, updatedAt time.Time) *Message {
	if content == nil {
		content = map[string]any{}
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	return &Message{
		id: id, conversationID: conversationID, seq: seq,
		senderType: senderType, senderID: senderID, clientMsgID: clientMsgID,
		contentType: contentType, content: content, replyToMsgID: replyToMsgID,
		status: status, serverTime: serverTime, metadata: metadata,
		createdAt: createdAt, updatedAt: updatedAt,
	}
}

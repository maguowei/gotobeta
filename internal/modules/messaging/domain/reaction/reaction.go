// Package reaction 是消息表情回应聚合。
//
// 聚合边界 = 包边界；reaction 是消息侧的附属数据，不占 timeline seq，
// 与 message 聚合解耦（跨聚合协调在应用层）。
package reaction

import (
	"time"
	"unicode/utf8"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// maxEmojiLen 是 emoji 字符串最大字节长度，与 Ent schema MaxLen 对齐。
const maxEmojiLen = 64

// Reaction 是表情回应聚合根。
type Reaction struct {
	id             int64
	conversationID int64
	messageID      int64
	userID         int64
	emoji          string
	createdAt      time.Time
}

func (r *Reaction) ID() int64             { return r.id }
func (r *Reaction) ConversationID() int64 { return r.conversationID }
func (r *Reaction) MessageID() int64      { return r.messageID }
func (r *Reaction) UserID() int64         { return r.userID }
func (r *Reaction) Emoji() string         { return r.emoji }
func (r *Reaction) CreatedAt() time.Time  { return r.createdAt }

// New 创建一条表情回应。
func New(id, conversationID, messageID, userID int64, emoji string) (*Reaction, error) {
	if emoji == "" {
		return nil, apperr.InvalidParam("emoji 不能为空")
	}
	if len(emoji) > maxEmojiLen || !utf8.ValidString(emoji) {
		return nil, apperr.InvalidParam("emoji 非法")
	}
	return &Reaction{
		id: id, conversationID: conversationID, messageID: messageID,
		userID: userID, emoji: emoji, createdAt: time.Now(),
	}, nil
}

// UnmarshalFromDB 从数据库重建表情回应聚合。
func UnmarshalFromDB(id, conversationID, messageID, userID int64, emoji string, createdAt time.Time) *Reaction {
	return &Reaction{
		id: id, conversationID: conversationID, messageID: messageID,
		userID: userID, emoji: emoji, createdAt: createdAt,
	}
}

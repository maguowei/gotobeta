// Package result 定义 messaging 模块用例出参（CQRS Result）。
package result

import "time"

// ConversationResult 是会话视图。
type ConversationResult struct {
	ID            int64      `json:"id"`
	WorkspaceID   int64      `json:"workspaceId"`
	Type          int8       `json:"type"`
	Visibility    int8       `json:"visibility"`
	Name          string     `json:"name"`
	Topic         string     `json:"topic"`
	CreatorID     int64      `json:"creatorId"`
	DMKey         *string    `json:"dmKey,omitempty"`
	LastSeq       int64      `json:"lastSeq"`
	LastMsgID     int64      `json:"lastMsgId"`
	LastMsgDigest string     `json:"lastMsgDigest"`
	LastMsgAt     *time.Time `json:"lastMsgAt,omitempty"`
	MemberCount   int        `json:"memberCount"`
	Status        int8       `json:"status"`
	CreatedAt     time.Time  `json:"createdAt"`
	// ReadSeq/Unread 仅在“我的会话列表”视图中填充（基于当前用户成员视图）。
	ReadSeq int64 `json:"readSeq"`
	Unread  int64 `json:"unread"`
}

// ConversationMemberResult 是会话成员视图。
type ConversationMemberResult struct {
	ConversationID int64     `json:"conversationId"`
	MemberType     int8      `json:"memberType"`
	MemberID       int64     `json:"memberId"`
	Role           int8      `json:"role"`
	ReadSeq        int64     `json:"readSeq"`
	IsMuted        bool      `json:"isMuted"`
	IsPinned       bool      `json:"isPinned"`
	Status         int8      `json:"status"`
	JoinedAt       time.Time `json:"joinedAt"`
}

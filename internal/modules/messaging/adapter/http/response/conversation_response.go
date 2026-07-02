// Package response 定义 messaging 模块的 HTTP 响应体，仅从 application 结果映射。
package response

import (
	"time"

	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
)

// ConversationResponse 是会话响应。
type ConversationResponse struct {
	ConversationID int64  `json:"conversationId,string"`
	WorkspaceID    int64  `json:"workspaceId,string"`
	Type           int8   `json:"type"`
	Visibility     int8   `json:"visibility"`
	Name           string `json:"name"`
	Topic          string `json:"topic"`
	CreatorID      int64  `json:"creatorId,string"`
	LastSeq        int64  `json:"lastSeq"`
	LastMsgID      int64  `json:"lastMsgId,string"`
	LastMsgDigest  string `json:"lastMsgDigest"`
	LastMsgAt      string `json:"lastMsgAt,omitempty"`
	MemberCount    int    `json:"memberCount"`
	ReadSeq        int64  `json:"readSeq"`
	Unread         int64  `json:"unread"`
	Status         int8   `json:"status"`
	CreatedAt      string `json:"createdAt"`
}

// ToConversationResponse 转换会话结果。
func ToConversationResponse(out *messagingresult.ConversationResult) ConversationResponse {
	resp := ConversationResponse{
		ConversationID: out.ID,
		WorkspaceID:    out.WorkspaceID,
		Type:           out.Type,
		Visibility:     out.Visibility,
		Name:           out.Name,
		Topic:          out.Topic,
		CreatorID:      out.CreatorID,
		LastSeq:        out.LastSeq,
		LastMsgID:      out.LastMsgID,
		LastMsgDigest:  out.LastMsgDigest,
		MemberCount:    out.MemberCount,
		ReadSeq:        out.ReadSeq,
		Unread:         out.Unread,
		Status:         out.Status,
		CreatedAt:      out.CreatedAt.Format(time.DateTime),
	}
	resp.LastMsgAt = formatNullableTime(out.LastMsgAt)
	return resp
}

// ToConversationListResponse 批量转换会话。
func ToConversationListResponse(items []*messagingresult.ConversationResult) []ConversationResponse {
	out := make([]ConversationResponse, 0, len(items))
	for _, item := range items {
		out = append(out, ToConversationResponse(item))
	}
	return out
}

// ConversationMemberResponse 是会话成员响应。
type ConversationMemberResponse struct {
	ConversationID int64  `json:"conversationId,string"`
	MemberType     int8   `json:"memberType"`
	MemberID       int64  `json:"memberId,string"`
	Role           int8   `json:"role"`
	ReadSeq        int64  `json:"readSeq"`
	IsMuted        bool   `json:"isMuted"`
	IsPinned       bool   `json:"isPinned"`
	Status         int8   `json:"status"`
	JoinedAt       string `json:"joinedAt"`
}

// ToConversationMemberResponse 转换会话成员结果。
func ToConversationMemberResponse(out *messagingresult.ConversationMemberResult) ConversationMemberResponse {
	return ConversationMemberResponse{
		ConversationID: out.ConversationID,
		MemberType:     out.MemberType,
		MemberID:       out.MemberID,
		Role:           out.Role,
		ReadSeq:        out.ReadSeq,
		IsMuted:        out.IsMuted,
		IsPinned:       out.IsPinned,
		Status:         out.Status,
		JoinedAt:       out.JoinedAt.Format(time.DateTime),
	}
}

// ToConversationMemberListResponse 批量转换会话成员。
func ToConversationMemberListResponse(items []*messagingresult.ConversationMemberResult) []ConversationMemberResponse {
	out := make([]ConversationMemberResponse, 0, len(items))
	for _, item := range items {
		out = append(out, ToConversationMemberResponse(item))
	}
	return out
}

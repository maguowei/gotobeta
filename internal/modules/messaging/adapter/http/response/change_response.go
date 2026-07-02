package response

import (
	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
)

// ChangeItem 是单条变更响应。
type ChangeItem struct {
	ChangeSeq  int64          `json:"changeSeq"`
	ChangeType int8           `json:"changeType"`
	MessageID  int64          `json:"messageId,string"`
	ActorID    int64          `json:"actorId,string"`
	Payload    map[string]any `json:"payload"`
}

// ChangesResponse 是变更流分页响应。
type ChangesResponse struct {
	Changes    []ChangeItem `json:"changes"`
	NextCursor int64        `json:"nextCursor"`
	HasMore    bool         `json:"hasMore"`
}

// ToChangesResponse 转换变更流结果页。
func ToChangesResponse(page *messagingresult.ChangesPage) ChangesResponse {
	items := make([]ChangeItem, 0, len(page.Changes))
	for _, c := range page.Changes {
		items = append(items, ChangeItem{
			ChangeSeq:  c.ChangeSeq,
			ChangeType: c.ChangeType,
			MessageID:  c.MessageID,
			ActorID:    c.ActorID,
			Payload:    c.Payload,
		})
	}
	return ChangesResponse{Changes: items, NextCursor: page.NextCursor, HasMore: page.HasMore}
}

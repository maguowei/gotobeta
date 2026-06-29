// Package imrt 定义 IM 实时网关所需的跨模块查询端口。
//
// realtime 模块据此读取会话成员以扇出推送，端口由 messaging 模块实现并经组合根注入，
// 避免 realtime 直接 import messaging（符合分层边界）。
package imrt

import "context"

// MemberLookup 查询会话成员，用于实时推送扇出。
type MemberLookup interface {
	// ConversationUserIDs 返回会话内全部活跃用户成员的 userID（不含 bot）。
	ConversationUserIDs(ctx context.Context, conversationID int64) ([]int64, error)
	// UserConversationPeers 返回与该用户共享任一会话的其他用户 userID 集合（presence 受众，不含自己）。
	UserConversationPeers(ctx context.Context, userID int64) ([]int64, error)
}

// ReadReporter 上报已读水位，供 realtime 处理 WS 上行 read 帧时回流到 messaging。
type ReadReporter interface {
	ReportRead(ctx context.Context, conversationID, userID, readSeq int64) error
}

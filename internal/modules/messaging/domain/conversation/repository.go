package conversation

import "context"

// WithMember 是会话与查询主体自身成员记录的组合视图。
type WithMember struct {
	Conversation *Conversation
	Member       *Member
}

// Repository 定义会话聚合的仓储接口。
type Repository interface {
	Create(ctx context.Context, c *Conversation) error
	FindByID(ctx context.Context, id int64) (*Conversation, error)
	// FindByDMKey 按单聊去重键查找，未命中返回 ErrNotFound。
	FindByDMKey(ctx context.Context, dmKey string) (*Conversation, error)
	Save(ctx context.Context, c *Conversation) error

	AddMember(ctx context.Context, m *Member) error
	FindMember(ctx context.Context, conversationID int64, memberType MemberType, memberID int64) (*Member, error)
	SaveMember(ctx context.Context, m *Member) error
	ListMembers(ctx context.Context, conversationID int64) ([]*Member, error)
	// ListByMember 返回某主体在指定工作区加入（活跃）的全部会话及其成员视图，按 last_msg_at 倒序。
	ListByMember(ctx context.Context, workspaceID int64, memberType MemberType, memberID int64) ([]WithMember, error)
	// ListActiveUserPeers 返回与该用户共享任一会话的其他活跃用户 ID 去重集。
	ListActiveUserPeers(ctx context.Context, userID int64) ([]int64, error)
}

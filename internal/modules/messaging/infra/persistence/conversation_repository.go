// Package persistence 是 messaging 模块的仓储 Ent 实现。
package persistence

import (
	"context"
	"log/slog"

	"github.com/maguowei/gotobeta/internal/ent"
	entconv "github.com/maguowei/gotobeta/internal/ent/conversation"
	entconvmember "github.com/maguowei/gotobeta/internal/ent/conversationmember"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/conversation"
)

// ConversationRepository 是会话仓储的 Ent 实现。
type ConversationRepository struct {
	client *ent.Client
	logger *slog.Logger
}

// NewConversationRepository 创建仓储。
func NewConversationRepository(client *ent.Client, logger *slog.Logger) *ConversationRepository {
	return &ConversationRepository{client: client, logger: logger}
}

// Create 保存会话。
func (r *ConversationRepository) Create(ctx context.Context, c *conversation.Conversation) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	create := client.Conversation.Create().
		SetBizID(c.ID()).
		SetWorkspaceID(c.WorkspaceID()).
		SetType(int8(c.Type())).
		SetVisibility(int8(c.Visibility())).
		SetName(c.Name()).
		SetTopic(c.Topic()).
		SetCreatorID(c.CreatorID()).
		SetLastSeq(c.LastSeq()).
		SetLastMsgID(c.LastMsgID()).
		SetLastMsgDigest(c.LastMsgDigest()).
		SetMemberCount(c.MemberCount()).
		SetStatus(int8(c.Status())).
		SetMetadata(c.Metadata()).
		SetCreatedAt(c.CreatedAt()).
		SetUpdatedAt(c.UpdatedAt())
	if c.DMKey() != nil {
		create.SetDmKey(*c.DMKey())
	}
	if c.LastMsgAt() != nil {
		create.SetLastMsgAt(*c.LastMsgAt())
	}
	if _, err := create.Save(ctx); err != nil {
		if ent.IsConstraintError(err) {
			return conversation.ErrDMExists
		}
		return err
	}
	return nil
}

// FindByID 按业务 ID 查找。
func (r *ConversationRepository) FindByID(ctx context.Context, id int64) (*conversation.Conversation, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.Conversation.Query().Where(entconv.BizID(id)).Only(ctx)
	if err != nil {
		return nil, mapEntNotFound(err, conversation.ErrNotFound)
	}
	return conversationToEntity(row), nil
}

// FindByDMKey 按单聊去重键查找。
func (r *ConversationRepository) FindByDMKey(ctx context.Context, dmKey string) (*conversation.Conversation, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.Conversation.Query().Where(entconv.DmKey(dmKey)).Only(ctx)
	if err != nil {
		return nil, mapEntNotFound(err, conversation.ErrNotFound)
	}
	return conversationToEntity(row), nil
}

// Save 更新会话可变字段。
func (r *ConversationRepository) Save(ctx context.Context, c *conversation.Conversation) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	update := client.Conversation.Update().
		Where(entconv.BizID(c.ID())).
		SetName(c.Name()).
		SetTopic(c.Topic()).
		SetVisibility(int8(c.Visibility())).
		SetLastSeq(c.LastSeq()).
		SetLastMsgID(c.LastMsgID()).
		SetLastMsgDigest(c.LastMsgDigest()).
		SetMemberCount(c.MemberCount()).
		SetStatus(int8(c.Status())).
		SetMetadata(c.Metadata()).
		SetUpdatedAt(c.UpdatedAt())
	if c.LastMsgAt() != nil {
		update.SetLastMsgAt(*c.LastMsgAt())
	}
	affected, err := update.Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return conversation.ErrNotFound
	}
	return nil
}

// AddMember 新增会话成员。
func (r *ConversationRepository) AddMember(ctx context.Context, m *conversation.Member) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	create := client.ConversationMember.Create().
		SetBizID(m.ID()).
		SetConversationID(m.ConversationID()).
		SetMemberType(int8(m.MemberType())).
		SetMemberID(m.MemberID()).
		SetRole(int8(m.Role())).
		SetReadSeq(m.ReadSeq()).
		SetIsMuted(m.IsMuted()).
		SetIsPinned(m.IsPinned()).
		SetStatus(int8(m.Status())).
		SetJoinedAt(m.JoinedAt()).
		SetCreatedAt(m.CreatedAt()).
		SetUpdatedAt(m.UpdatedAt())
	if m.LastReadAt() != nil {
		create.SetLastReadAt(*m.LastReadAt())
	}
	if _, err := create.Save(ctx); err != nil {
		return err
	}
	return nil
}

// FindMember 查找会话成员。
func (r *ConversationRepository) FindMember(ctx context.Context, conversationID int64, memberType conversation.MemberType, memberID int64) (*conversation.Member, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.ConversationMember.Query().
		Where(
			entconvmember.ConversationID(conversationID),
			entconvmember.MemberType(int8(memberType)),
			entconvmember.MemberID(memberID),
		).Only(ctx)
	if err != nil {
		return nil, mapEntNotFound(err, conversation.ErrMemberNotFound)
	}
	return memberToEntity(row), nil
}

// SaveMember 更新会话成员可变字段。
func (r *ConversationRepository) SaveMember(ctx context.Context, m *conversation.Member) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	update := client.ConversationMember.Update().
		Where(entconvmember.BizID(m.ID())).
		SetRole(int8(m.Role())).
		SetReadSeq(m.ReadSeq()).
		SetIsMuted(m.IsMuted()).
		SetIsPinned(m.IsPinned()).
		SetStatus(int8(m.Status())).
		SetUpdatedAt(m.UpdatedAt())
	if m.LastReadAt() != nil {
		update.SetLastReadAt(*m.LastReadAt())
	}
	affected, err := update.Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return conversation.ErrMemberNotFound
	}
	return nil
}

// ListMembers 列出会话全部成员。
func (r *ConversationRepository) ListMembers(ctx context.Context, conversationID int64) ([]*conversation.Member, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	rows, err := client.ConversationMember.Query().
		Where(entconvmember.ConversationID(conversationID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*conversation.Member, 0, len(rows))
	for _, row := range rows {
		items = append(items, memberToEntity(row))
	}
	return items, nil
}

// ListByMember 返回某主体在指定工作区加入（活跃）的全部会话及其成员视图，按 last_msg_at 倒序。
func (r *ConversationRepository) ListByMember(ctx context.Context, workspaceID int64, memberType conversation.MemberType, memberID int64) ([]conversation.WithMember, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	memberRows, err := client.ConversationMember.Query().
		Where(
			entconvmember.MemberType(int8(memberType)),
			entconvmember.MemberID(memberID),
			entconvmember.StatusEQ(int8(conversation.MemberActive)),
		).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(memberRows) == 0 {
		return []conversation.WithMember{}, nil
	}
	convIDs := make([]int64, 0, len(memberRows))
	memberByConv := make(map[int64]*ent.ConversationMember, len(memberRows))
	for _, m := range memberRows {
		convIDs = append(convIDs, m.ConversationID)
		memberByConv[m.ConversationID] = m
	}
	rows, err := client.Conversation.Query().
		Where(entconv.BizIDIn(convIDs...), entconv.WorkspaceID(workspaceID)).
		Order(ent.Desc(entconv.FieldLastMsgAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]conversation.WithMember, 0, len(rows))
	for _, row := range rows {
		items = append(items, conversation.WithMember{
			Conversation: conversationToEntity(row),
			Member:       memberToEntity(memberByConv[row.BizID]),
		})
	}
	return items, nil
}

// ListActiveUserPeers 返回与该用户共享任一会话的其他活跃用户 ID 去重集（两次往返，避免逐会话 N+1）。
func (r *ConversationRepository) ListActiveUserPeers(ctx context.Context, userID int64) ([]int64, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	memberRows, err := client.ConversationMember.Query().
		Where(
			entconvmember.MemberType(int8(conversation.MemberUser)),
			entconvmember.MemberID(userID),
			entconvmember.StatusEQ(int8(conversation.MemberActive)),
		).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(memberRows) == 0 {
		return []int64{}, nil
	}
	convIDs := make([]int64, 0, len(memberRows))
	for _, m := range memberRows {
		convIDs = append(convIDs, m.ConversationID)
	}
	peerRows, err := client.ConversationMember.Query().
		Where(
			entconvmember.ConversationIDIn(convIDs...),
			entconvmember.MemberType(int8(conversation.MemberUser)),
			entconvmember.MemberIDNEQ(userID),
			entconvmember.StatusEQ(int8(conversation.MemberActive)),
		).All(ctx)
	if err != nil {
		return nil, err
	}
	seen := make(map[int64]struct{}, len(peerRows))
	ids := make([]int64, 0, len(peerRows))
	for _, m := range peerRows {
		if _, ok := seen[m.MemberID]; ok {
			continue
		}
		seen[m.MemberID] = struct{}{}
		ids = append(ids, m.MemberID)
	}
	return ids, nil
}

func conversationToEntity(row *ent.Conversation) *conversation.Conversation {
	var dmKey *string
	if row.DmKey != nil {
		dmKey = row.DmKey
	}
	return conversation.UnmarshalFromDB(
		row.BizID, row.WorkspaceID, conversation.Type(row.Type), conversation.Visibility(row.Visibility),
		row.Name, row.Topic, row.CreatorID, dmKey,
		row.LastSeq, row.LastMsgID, row.LastMsgDigest, row.LastMsgAt,
		row.MemberCount, conversation.Status(row.Status), row.Metadata,
		row.CreatedAt, row.UpdatedAt,
	)
}

func memberToEntity(row *ent.ConversationMember) *conversation.Member {
	return conversation.UnmarshalMemberFromDB(
		row.BizID, row.ConversationID, conversation.MemberType(row.MemberType), row.MemberID,
		conversation.Role(row.Role), row.ReadSeq, row.LastReadAt, row.IsMuted, row.IsPinned,
		conversation.MemberStatus(row.Status), row.JoinedAt, row.CreatedAt, row.UpdatedAt,
	)
}

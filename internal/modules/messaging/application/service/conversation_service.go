// Package service 编排 messaging 模块用例（会话、成员、消息）。
package service

import (
	"context"
	stderrors "errors"
	"log/slog"

	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/conversation"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
	"github.com/maguowei/gotobeta/internal/pkg/idgen"
	"github.com/maguowei/gotobeta/internal/pkg/persistence"
	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
)

// 工作区级权限动作编码，必须与 workspace 平台权限 seed 保持一致。
const (
	actionChannelCreate    = "channel.create"
	actionConversationRead = "conversation.read"
)

// ConversationService 编排会话相关用例。
type ConversationService struct {
	conversations conversation.Repository
	checker       authz.Checker
	idGenerator   idgen.Generator
	txRunner      persistence.TxRunner
	logger        *slog.Logger
}

// NewConversationService 创建服务。
func NewConversationService(
	conversations conversation.Repository,
	checker authz.Checker,
	idGenerator idgen.Generator,
	txRunner persistence.TxRunner,
	logger *slog.Logger,
) *ConversationService {
	return &ConversationService{
		conversations: conversations,
		checker:       checker,
		idGenerator:   idGenerator,
		txRunner:      txRunner,
		logger:        logger,
	}
}

func toConversationResult(c *conversation.Conversation) *messagingresult.ConversationResult {
	return &messagingresult.ConversationResult{
		ID:            c.ID(),
		WorkspaceID:   c.WorkspaceID(),
		Type:          int8(c.Type()),
		Visibility:    int8(c.Visibility()),
		Name:          c.Name(),
		Topic:         c.Topic(),
		CreatorID:     c.CreatorID(),
		DMKey:         c.DMKey(),
		LastSeq:       c.LastSeq(),
		LastMsgID:     c.LastMsgID(),
		LastMsgDigest: c.LastMsgDigest(),
		LastMsgAt:     c.LastMsgAt(),
		MemberCount:   c.MemberCount(),
		Status:        int8(c.Status()),
		CreatedAt:     c.CreatedAt(),
	}
}

func toConversationMemberResult(m *conversation.Member) *messagingresult.ConversationMemberResult {
	return &messagingresult.ConversationMemberResult{
		ConversationID: m.ConversationID(),
		MemberType:     int8(m.MemberType()),
		MemberID:       m.MemberID(),
		Role:           int8(m.Role()),
		ReadSeq:        m.ReadSeq(),
		IsMuted:        m.IsMuted(),
		IsPinned:       m.IsPinned(),
		Status:         int8(m.Status()),
		JoinedAt:       m.JoinedAt(),
	}
}

// requireActiveMembership 校验用户是会话的活跃成员，返回其成员记录。
func (s *ConversationService) requireActiveMembership(ctx context.Context, convID int64, userID int64) (*conversation.Member, error) {
	mem, err := s.conversations.FindMember(ctx, convID, conversation.MemberUser, userID)
	if err != nil {
		if stderrors.Is(err, conversation.ErrMemberNotFound) {
			return nil, apperr.Forbidden("不是该会话成员")
		}
		return nil, wrapInfrastructureError("查询会话成员失败", err)
	}
	if mem.Status() != conversation.MemberActive {
		return nil, apperr.Forbidden("不是该会话成员")
	}
	return mem, nil
}

func wrapInfrastructureError(message string, err error) error {
	if stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return apperr.Internal(message, err)
}

// assertWorkspaceScope 是 DataScope 纵深防御第二层：确认 ctx 中受信工作区
// （由 WorkspaceScope 中间件从 path 注入）与命令携带的工作区一致，不一致即越权。
// ctx 未注入工作区时（如内部调用/测试）跳过，由其它层兜底。
func assertWorkspaceScope(ctx context.Context, cmdWorkspaceID int64) error {
	if ctxWS, ok := requestctx.WorkspaceID(ctx); ok && ctxWS != cmdWorkspaceID {
		return apperr.Forbidden("工作区不一致")
	}
	return nil
}

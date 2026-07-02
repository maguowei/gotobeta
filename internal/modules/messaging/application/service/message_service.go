package service

import (
	"context"
	stderrors "errors"
	"log/slog"
	"time"

	messagingport "github.com/maguowei/gotobeta/internal/modules/messaging/application/port"
	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/conversation"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/message"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/messagechange"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/reaction"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
	"github.com/maguowei/gotobeta/internal/pkg/event"
	"github.com/maguowei/gotobeta/internal/pkg/idgen"
	"github.com/maguowei/gotobeta/internal/pkg/persistence"
)

// MessageService 编排消息相关用例（发送、拉取、撤回）。
type MessageService struct {
	messages      message.Repository
	conversations conversation.Repository
	reactions     reaction.Repository
	changes       messagechange.Repository
	seqAllocator  messagingport.SeqAllocator
	checker       authz.Checker
	publisher     event.Publisher
	idGenerator   idgen.Generator
	txRunner      persistence.TxRunner
	recallWindow  time.Duration
	pageSize      int
	maxPageSize   int
	logger        *slog.Logger
	metrics       messagingport.MessageMetrics
}

// NewMessageService 创建服务。metrics 可为 nil（不埋点）。
func NewMessageService(
	messages message.Repository,
	conversations conversation.Repository,
	reactions reaction.Repository,
	changes messagechange.Repository,
	seqAllocator messagingport.SeqAllocator,
	checker authz.Checker,
	publisher event.Publisher,
	idGenerator idgen.Generator,
	txRunner persistence.TxRunner,
	recallWindow time.Duration,
	pageSize int,
	logger *slog.Logger,
	metrics messagingport.MessageMetrics,
) *MessageService {
	if pageSize <= 0 {
		pageSize = 50
	}
	return &MessageService{
		messages:      messages,
		conversations: conversations,
		reactions:     reactions,
		changes:       changes,
		seqAllocator:  seqAllocator,
		checker:       checker,
		publisher:     publisher,
		idGenerator:   idGenerator,
		txRunner:      txRunner,
		recallWindow:  recallWindow,
		pageSize:      pageSize,
		maxPageSize:   200,
		logger:        logger,
		metrics:       metrics,
	}
}

func toMessageResult(m *message.Message) *messagingresult.MessageResult {
	return &messagingresult.MessageResult{
		MessageID:      m.ID(),
		ConversationID: m.ConversationID(),
		Seq:            m.Seq(),
		SenderType:     int8(m.SenderType()),
		SenderID:       m.SenderID(),
		ContentType:    int8(m.ContentType()),
		Content:        m.Content(),
		ReplyToMsgID:   m.ReplyToMsgID(),
		Status:         int8(m.Status()),
		ServerTime:     m.ServerTime(),
		EditedAt:       m.EditedAt(),
	}
}

// appendChange 生成变更 ID 并把一条 messagechange 追加进变更流（须在事务内调用）。
func (s *MessageService) appendChange(txCtx context.Context, convID, seq int64, ct messagechange.ChangeType, targetID, operatorID int64, payload map[string]any) error {
	changeID, err := s.idGenerator.NextID(txCtx)
	if err != nil {
		return apperr.WrapInternal("生成变更 ID 失败", err)
	}
	chg, err := messagechange.New(changeID, convID, seq, ct, targetID, operatorID, payload)
	if err != nil {
		return err
	}
	if err := s.changes.Append(txCtx, chg); err != nil {
		return apperr.WrapInternal("追加变更流失败", err)
	}
	return nil
}

// requireActiveMember 校验用户是会话活跃成员，返回成员记录。
func (s *MessageService) requireActiveMember(ctx context.Context, convID, userID int64) (*conversation.Member, error) {
	mem, err := s.conversations.FindMember(ctx, convID, conversation.MemberUser, userID)
	if err != nil {
		if stderrors.Is(err, conversation.ErrMemberNotFound) {
			return nil, apperr.Forbidden("不是该会话成员")
		}
		return nil, apperr.WrapInternal("查询会话成员失败", err)
	}
	if mem.Status() != conversation.MemberActive {
		return nil, apperr.Forbidden("不是该会话成员")
	}
	return mem, nil
}

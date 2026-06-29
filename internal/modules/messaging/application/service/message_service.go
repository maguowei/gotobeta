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
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
	"github.com/maguowei/gotobeta/internal/pkg/event"
	"github.com/maguowei/gotobeta/internal/pkg/idgen"
	"github.com/maguowei/gotobeta/internal/pkg/persistence"
)

// 工作区级权限动作编码，必须与 workspace 平台权限 seed 保持一致。
const actionMessageRecall = "message.recall"

// MessageService 编排消息相关用例（发送、拉取、撤回）。
type MessageService struct {
	messages      message.Repository
	conversations conversation.Repository
	seqAllocator  messagingport.SeqAllocator
	checker       authz.Checker
	publisher     event.Publisher
	idGenerator   idgen.Generator
	txRunner      persistence.TxRunner
	recallWindow  time.Duration
	pageSize      int
	maxPageSize   int
	logger        *slog.Logger
}

// NewMessageService 创建服务。
func NewMessageService(
	messages message.Repository,
	conversations conversation.Repository,
	seqAllocator messagingport.SeqAllocator,
	checker authz.Checker,
	publisher event.Publisher,
	idGenerator idgen.Generator,
	txRunner persistence.TxRunner,
	recallWindow time.Duration,
	pageSize int,
	logger *slog.Logger,
) *MessageService {
	if pageSize <= 0 {
		pageSize = 50
	}
	return &MessageService{
		messages:      messages,
		conversations: conversations,
		seqAllocator:  seqAllocator,
		checker:       checker,
		publisher:     publisher,
		idGenerator:   idGenerator,
		txRunner:      txRunner,
		recallWindow:  recallWindow,
		pageSize:      pageSize,
		maxPageSize:   200,
		logger:        logger,
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
	}
}

// requireActiveMember 校验用户是会话活跃成员，返回成员记录。
func (s *MessageService) requireActiveMember(ctx context.Context, convID, userID int64) (*conversation.Member, error) {
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

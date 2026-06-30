package service

import (
	"context"
	stderrors "errors"
	"log/slog"
	"time"

	messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"
	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/conversation"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/message"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
	"github.com/maguowei/gotobeta/internal/pkg/imevent"
	loggerx "github.com/maguowei/gotobeta/internal/pkg/logger"
)

// SendMessage 发送消息：成员校验 → 幂等 → 事务内分配 seq 落库并更新会话游标 → 发布事件。
func (s *MessageService) SendMessage(ctx context.Context, cmd messagingcmd.SendMessageCommand) (*messagingresult.MessageResult, error) {
	start := time.Now()
	if err := assertWorkspaceScope(ctx, cmd.WorkspaceID); err != nil {
		return nil, err
	}
	if cmd.ClientMsgID == "" {
		return nil, apperr.InvalidParam("client_msg_id 不能为空")
	}
	conv, err := s.conversations.FindByID(ctx, cmd.ConversationID)
	if err != nil {
		if stderrors.Is(err, conversation.ErrNotFound) {
			return nil, apperr.NotFound("会话不存在")
		}
		return nil, wrapInfrastructureError("查询会话失败", err)
	}
	if conv.Status() != conversation.StatusActive {
		return nil, apperr.InvalidParam("会话已归档或解散")
	}
	mem, err := s.requireActiveMember(ctx, cmd.ConversationID, cmd.SenderUserID)
	if err != nil {
		return nil, err
	}
	if mem.IsMuted() {
		return nil, apperr.Forbidden("已被禁言")
	}

	// 引用回复校验：被引用消息必须存在且属于同一会话。
	if cmd.ReplyToMsgID > 0 {
		replied, err := s.messages.FindByID(ctx, cmd.ReplyToMsgID)
		if err != nil {
			if stderrors.Is(err, message.ErrNotFound) {
				return nil, apperr.InvalidParam("被引用的消息不存在")
			}
			return nil, wrapInfrastructureError("查询被引用消息失败", err)
		}
		if replied.ConversationID() != cmd.ConversationID {
			return nil, apperr.InvalidParam("被引用的消息不属于该会话")
		}
	}

	// 幂等：命中相同 client_msg_id 直接返回原结果。
	if existing, err := s.messages.FindByClientMsgID(ctx, cmd.ConversationID, cmd.ClientMsgID); err == nil {
		return toMessageResult(existing), nil
	} else if !stderrors.Is(err, message.ErrNotFound) {
		return nil, wrapInfrastructureError("查询幂等消息失败", err)
	}

	msgID, err := s.idGenerator.NextID(ctx)
	if err != nil {
		return nil, wrapInfrastructureError("生成消息 ID 失败", err)
	}

	var msg *message.Message
	err = s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		seqStart := time.Now()
		seq, err := s.seqAllocator.Next(txCtx, cmd.ConversationID)
		if s.metrics != nil {
			s.metrics.ObserveSeqAlloc(txCtx, time.Since(seqStart))
		}
		if err != nil {
			return wrapInfrastructureError("分配 seq 失败", err)
		}
		m, err := message.New(msgID, cmd.ConversationID, seq, message.SenderUser, cmd.SenderUserID,
			cmd.ClientMsgID, message.ContentType(cmd.ContentType), cmd.Content, cmd.ReplyToMsgID)
		if err != nil {
			return err
		}
		if err := s.messages.Create(txCtx, m); err != nil {
			return wrapInfrastructureError("保存消息失败", err)
		}
		conv.ApplyMessage(seq, msgID, m.Digest(), m.ServerTime())
		if err := s.conversations.Save(txCtx, conv); err != nil {
			return wrapInfrastructureError("更新会话游标失败", err)
		}
		msg = m
		return nil
	})
	if err != nil {
		loggerx.WithError(ctx, s.logger, "send message failed", err, slog.Int64("conversationId", cmd.ConversationID))
		return nil, err
	}

	s.publishCreated(ctx, conv.WorkspaceID(), msg)
	if s.metrics != nil {
		s.metrics.ObserveMessageLatency(ctx, time.Since(start))
	}
	s.logger.InfoContext(ctx, "message sent", slog.Int64("conversationId", cmd.ConversationID), slog.Int64("messageId", msgID), slog.Int64("seq", msg.Seq()))
	return toMessageResult(msg), nil
}

// RecallMessage 撤回消息：本人在窗口内或具 message.recall 权限可撤回，并写入系统撤回条目。
func (s *MessageService) RecallMessage(ctx context.Context, cmd messagingcmd.RecallMessageCommand) error {
	if err := assertWorkspaceScope(ctx, cmd.WorkspaceID); err != nil {
		return err
	}
	msg, err := s.messages.FindByID(ctx, cmd.MessageID)
	if err != nil {
		if stderrors.Is(err, message.ErrNotFound) {
			return apperr.NotFound("消息不存在")
		}
		return wrapInfrastructureError("查询消息失败", err)
	}
	if msg.ConversationID() != cmd.ConversationID {
		return apperr.InvalidParam("消息不属于该会话")
	}
	if _, err := s.requireActiveMember(ctx, cmd.ConversationID, cmd.OperatorUserID); err != nil {
		return err
	}

	// 非本人撤回需要 message.recall 权限。
	isSelf := msg.SenderType() == message.SenderUser && msg.SenderID() == cmd.OperatorUserID
	if !isSelf {
		if err := s.checker.Check(ctx, authz.Request{
			WorkspaceID: cmd.WorkspaceID,
			Subject:     authz.Subject{UserID: cmd.OperatorUserID},
			Action:      actionMessageRecall,
		}); err != nil {
			return err
		}
	}

	if err := msg.Recall(time.Now(), s.recallWindow); err != nil {
		if stderrors.Is(err, message.ErrRecallWindowExpired) {
			return apperr.InvalidParam("已超过撤回时间窗口")
		}
		if stderrors.Is(err, message.ErrNotRecallable) {
			return apperr.InvalidParam("该消息不可撤回")
		}
		return err
	}

	conv, err := s.conversations.FindByID(ctx, cmd.ConversationID)
	if err != nil {
		return wrapInfrastructureError("查询会话失败", err)
	}
	sysID, err := s.idGenerator.NextID(ctx)
	if err != nil {
		return wrapInfrastructureError("生成系统消息 ID 失败", err)
	}

	var sysMsg *message.Message
	err = s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.messages.Save(txCtx, msg); err != nil {
			return wrapInfrastructureError("更新消息状态失败", err)
		}
		seq, err := s.seqAllocator.Next(txCtx, cmd.ConversationID)
		if err != nil {
			return wrapInfrastructureError("分配 seq 失败", err)
		}
		sys := message.NewSystem(sysID, cmd.ConversationID, seq, message.ContentRecall, map[string]any{
			"recalledMsgId": msg.ID(),
			"operatorId":    cmd.OperatorUserID,
		})
		if err := s.messages.Create(txCtx, sys); err != nil {
			return wrapInfrastructureError("保存撤回条目失败", err)
		}
		conv.ApplyMessage(seq, sysID, sys.Digest(), sys.ServerTime())
		if err := s.conversations.Save(txCtx, conv); err != nil {
			return wrapInfrastructureError("更新会话游标失败", err)
		}
		sysMsg = sys
		return nil
	})
	if err != nil {
		loggerx.WithError(ctx, s.logger, "recall message failed", err, slog.Int64("messageId", cmd.MessageID))
		return err
	}

	s.publishCreated(ctx, conv.WorkspaceID(), sysMsg)
	s.logger.InfoContext(ctx, "message recalled", slog.Int64("conversationId", cmd.ConversationID), slog.Int64("messageId", cmd.MessageID))
	return nil
}

// EditMessage 编辑消息：仅本人在编辑窗口内可原地更新文本内容，并发布编辑事件供在线端同步。
func (s *MessageService) EditMessage(ctx context.Context, cmd messagingcmd.EditMessageCommand) (*messagingresult.MessageResult, error) {
	if err := assertWorkspaceScope(ctx, cmd.WorkspaceID); err != nil {
		return nil, err
	}
	msg, err := s.messages.FindByID(ctx, cmd.MessageID)
	if err != nil {
		if stderrors.Is(err, message.ErrNotFound) {
			return nil, apperr.NotFound("消息不存在")
		}
		return nil, wrapInfrastructureError("查询消息失败", err)
	}
	if msg.ConversationID() != cmd.ConversationID {
		return nil, apperr.InvalidParam("消息不属于该会话")
	}
	if _, err := s.requireActiveMember(ctx, cmd.ConversationID, cmd.OperatorUserID); err != nil {
		return nil, err
	}

	// 仅消息发送者本人可编辑。
	isSelf := msg.SenderType() == message.SenderUser && msg.SenderID() == cmd.OperatorUserID
	if !isSelf {
		return nil, apperr.Forbidden("只能编辑自己的消息")
	}

	if err := msg.Edit(cmd.Content, time.Now(), s.recallWindow); err != nil {
		if stderrors.Is(err, message.ErrEditWindowExpired) {
			return nil, apperr.InvalidParam("已超过编辑时间窗口")
		}
		if stderrors.Is(err, message.ErrNotEditable) {
			return nil, apperr.InvalidParam("该消息不可编辑")
		}
		return nil, err
	}

	if err := s.messages.Save(ctx, msg); err != nil {
		return nil, wrapInfrastructureError("保存编辑内容失败", err)
	}

	workspaceID := cmd.WorkspaceID
	evt := imevent.NewMessageEditedEvent(workspaceID, msg.ConversationID(), msg.ID(), msg.Content(), *msg.EditedAt())
	if err := s.publisher.Publish(ctx, evt); err != nil {
		loggerx.WithError(ctx, s.logger, "publish message edited event failed", err, slog.Int64("messageId", msg.ID()))
	}
	s.logger.InfoContext(ctx, "message edited", slog.Int64("conversationId", cmd.ConversationID), slog.Int64("messageId", cmd.MessageID))
	return toMessageResult(msg), nil
}

// ReportRead 上报已读水位（单调推进 read_seq），并发布已读更新事件供多端对齐。
func (s *MessageService) ReportRead(ctx context.Context, cmd messagingcmd.ReportReadCommand) error {
	mem, err := s.requireActiveMember(ctx, cmd.ConversationID, cmd.UserID)
	if err != nil {
		return err
	}
	if !mem.MarkRead(cmd.ReadSeq, time.Now()) {
		// 水位未推进（旧值不回退），幂等返回。
		return nil
	}
	if err := s.conversations.SaveMember(ctx, mem); err != nil {
		return wrapInfrastructureError("更新已读水位失败", err)
	}
	workspaceID := int64(0)
	if conv, err := s.conversations.FindByID(ctx, cmd.ConversationID); err == nil {
		workspaceID = conv.WorkspaceID()
	}
	evt := imevent.NewReadUpdatedEvent(workspaceID, cmd.ConversationID, cmd.UserID, mem.ReadSeq(), time.Now())
	if err := s.publisher.Publish(ctx, evt); err != nil {
		loggerx.WithError(ctx, s.logger, "publish read updated event failed", err, slog.Int64("conversationId", cmd.ConversationID))
	}
	return nil
}

// publishCreated 在事务提交后尽力发布消息创建事件（跨模块共享契约）。
func (s *MessageService) publishCreated(ctx context.Context, workspaceID int64, m *message.Message) {
	evt := imevent.NewMessageCreatedEvent(workspaceID, m.ConversationID(), m.ID(), m.Seq(), int8(m.SenderType()), m.SenderID(), int8(m.ContentType()), m.ServerTime())
	if err := s.publisher.Publish(ctx, evt); err != nil {
		loggerx.WithError(ctx, s.logger, "publish message created event failed", err, slog.Int64("messageId", m.ID()))
	}
}

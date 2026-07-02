package service

import (
	"context"
	stderrors "errors"
	"log/slog"
	"time"

	messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/message"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/messagechange"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/reaction"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
	"github.com/maguowei/gotobeta/internal/pkg/imevent"
	loggerx "github.com/maguowei/gotobeta/internal/pkg/logger"
)

// 工作区级权限动作编码，必须与 workspace 平台权限 seed 保持一致。
const actionMessageReact = "message.react"

// AddReaction 给消息添加表情回应：成员校验 + message.react 权限 → 唯一约束幂等落库 → 发布事件。
func (s *MessageService) AddReaction(ctx context.Context, cmd messagingcmd.AddReactionCommand) error {
	if err := assertWorkspaceScope(ctx, cmd.WorkspaceID); err != nil {
		return err
	}
	if cmd.Emoji == "" {
		return apperr.InvalidParam("emoji 不能为空")
	}
	if err := s.assertMessageInConversation(ctx, cmd.MessageID, cmd.ConversationID); err != nil {
		return err
	}
	if _, err := s.requireActiveMember(ctx, cmd.ConversationID, cmd.OperatorUserID); err != nil {
		return err
	}
	if err := s.checker.Check(ctx, authz.Request{
		WorkspaceID: cmd.WorkspaceID,
		Subject:     authz.Subject{UserID: cmd.OperatorUserID},
		Action:      actionMessageReact,
	}); err != nil {
		return err
	}

	id, err := s.idGenerator.NextID(ctx)
	if err != nil {
		return wrapInfrastructureError("生成表情回应 ID 失败", err)
	}
	rc, err := reaction.New(id, cmd.ConversationID, cmd.MessageID, cmd.OperatorUserID, cmd.Emoji)
	if err != nil {
		return err
	}
	err = s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.reactions.Add(txCtx, rc); err != nil {
			return err // 含 reaction.ErrAlreadyExists，外层判定
		}
		seq, err := s.seqAllocator.Next(txCtx, cmd.ConversationID)
		if err != nil {
			return wrapInfrastructureError("分配 seq 失败", err)
		}
		changeID, err := s.idGenerator.NextID(txCtx)
		if err != nil {
			return wrapInfrastructureError("生成变更 ID 失败", err)
		}
		chg, err := messagechange.New(changeID, cmd.ConversationID, seq, messagechange.ChangeReactionAdd, cmd.MessageID, cmd.OperatorUserID, map[string]any{
			"userId": cmd.OperatorUserID,
			"emoji":  cmd.Emoji,
		})
		if err != nil {
			return err
		}
		return s.changes.Append(txCtx, chg)
	})
	if err != nil {
		if stderrors.Is(err, reaction.ErrAlreadyExists) {
			return nil // 幂等：已回应过，no-op 不发事件、不写变更（事务已回滚）
		}
		return wrapInfrastructureError("保存表情回应失败", err)
	}

	s.publishReaction(ctx, cmd.WorkspaceID, cmd.ConversationID, cmd.MessageID, cmd.OperatorUserID, cmd.Emoji, imevent.ReactionActionAdd)
	s.logger.InfoContext(ctx, "reaction added", slog.Int64("messageId", cmd.MessageID), slog.Int64("userId", cmd.OperatorUserID))
	return nil
}

// RemoveReaction 取消本人对消息的表情回应；未回应过则幂等返回不发事件。
func (s *MessageService) RemoveReaction(ctx context.Context, cmd messagingcmd.RemoveReactionCommand) error {
	if err := assertWorkspaceScope(ctx, cmd.WorkspaceID); err != nil {
		return err
	}
	if cmd.Emoji == "" {
		return apperr.InvalidParam("emoji 不能为空")
	}
	if err := s.assertMessageInConversation(ctx, cmd.MessageID, cmd.ConversationID); err != nil {
		return err
	}
	if _, err := s.requireActiveMember(ctx, cmd.ConversationID, cmd.OperatorUserID); err != nil {
		return err
	}

	var removed bool
	err := s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		var rerr error
		removed, rerr = s.reactions.Remove(txCtx, cmd.MessageID, cmd.OperatorUserID, cmd.Emoji)
		if rerr != nil {
			return wrapInfrastructureError("删除表情回应失败", rerr)
		}
		if !removed {
			return nil // 未回应过，幂等 no-op（不写变更）
		}
		seq, err := s.seqAllocator.Next(txCtx, cmd.ConversationID)
		if err != nil {
			return wrapInfrastructureError("分配 seq 失败", err)
		}
		changeID, err := s.idGenerator.NextID(txCtx)
		if err != nil {
			return wrapInfrastructureError("生成变更 ID 失败", err)
		}
		chg, err := messagechange.New(changeID, cmd.ConversationID, seq, messagechange.ChangeReactionRemove, cmd.MessageID, cmd.OperatorUserID, map[string]any{
			"userId": cmd.OperatorUserID,
			"emoji":  cmd.Emoji,
		})
		if err != nil {
			return err
		}
		return s.changes.Append(txCtx, chg)
	})
	if err != nil {
		return err
	}
	if !removed {
		return nil
	}

	s.publishReaction(ctx, cmd.WorkspaceID, cmd.ConversationID, cmd.MessageID, cmd.OperatorUserID, cmd.Emoji, imevent.ReactionActionRemove)
	s.logger.InfoContext(ctx, "reaction removed", slog.Int64("messageId", cmd.MessageID), slog.Int64("userId", cmd.OperatorUserID))
	return nil
}

// assertMessageInConversation 校验消息存在且属于指定会话。
func (s *MessageService) assertMessageInConversation(ctx context.Context, messageID, conversationID int64) error {
	msg, err := s.messages.FindByID(ctx, messageID)
	if err != nil {
		if stderrors.Is(err, message.ErrNotFound) {
			return apperr.NotFound("消息不存在")
		}
		return wrapInfrastructureError("查询消息失败", err)
	}
	if msg.ConversationID() != conversationID {
		return apperr.InvalidParam("消息不属于该会话")
	}
	return nil
}

// publishReaction 尽力发布表情回应变更事件（跨模块共享契约）。
func (s *MessageService) publishReaction(ctx context.Context, workspaceID, conversationID, messageID, userID int64, emoji string, action int8) {
	evt := imevent.NewReactionUpdatedEvent(workspaceID, conversationID, messageID, userID, emoji, action, time.Now())
	if err := s.publisher.Publish(ctx, evt); err != nil {
		loggerx.WithError(ctx, s.logger, "publish reaction updated event failed", err, slog.Int64("messageId", messageID))
	}
}

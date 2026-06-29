package service

import (
	"context"
	stderrors "errors"
	"log/slog"

	messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"
	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/conversation"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
	loggerx "github.com/maguowei/gotobeta/internal/pkg/logger"
)

// CreateConversation 创建会话/频道；单聊按 dm_key 幂等去重，命中已有则直接返回。
func (s *ConversationService) CreateConversation(ctx context.Context, cmd messagingcmd.CreateConversationCommand) (*messagingresult.ConversationResult, error) {
	if err := assertWorkspaceScope(ctx, cmd.WorkspaceID); err != nil {
		return nil, err
	}
	switch conversation.Type(cmd.Type) {
	case conversation.TypeDM:
		return s.createDM(ctx, cmd)
	case conversation.TypeGroup, conversation.TypeChannel:
		return s.createGroupOrChannel(ctx, cmd)
	default:
		return nil, apperr.InvalidParam("会话类型非法")
	}
}

func (s *ConversationService) createDM(ctx context.Context, cmd messagingcmd.CreateConversationCommand) (*messagingresult.ConversationResult, error) {
	if cmd.PeerUserID == 0 || cmd.PeerUserID == cmd.OperatorUserID {
		return nil, apperr.InvalidParam("单聊对端非法")
	}
	// 任一工作区成员都可发起单聊；用 conversation.read 校验成员身份。
	if err := s.checker.Check(ctx, authz.Request{
		WorkspaceID: cmd.WorkspaceID,
		Subject:     authz.Subject{UserID: cmd.OperatorUserID},
		Action:      actionConversationRead,
	}); err != nil {
		return nil, err
	}

	dmKey := conversation.DMKey(cmd.WorkspaceID, cmd.OperatorUserID, cmd.PeerUserID)
	if existing, err := s.conversations.FindByDMKey(ctx, dmKey); err == nil {
		return toConversationResult(existing), nil
	} else if !stderrors.Is(err, conversation.ErrNotFound) {
		return nil, wrapInfrastructureError("查询单聊失败", err)
	}

	convID, err := s.idGenerator.NextID(ctx)
	if err != nil {
		return nil, wrapInfrastructureError("生成会话 ID 失败", err)
	}
	conv, err := conversation.NewDM(convID, cmd.WorkspaceID, cmd.OperatorUserID, cmd.PeerUserID, cmd.OperatorUserID)
	if err != nil {
		return nil, err
	}

	err = s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.conversations.Create(txCtx, conv); err != nil {
			if stderrors.Is(err, conversation.ErrDMExists) {
				return apperr.Conflict("单聊已存在")
			}
			return wrapInfrastructureError("保存会话失败", err)
		}
		for _, uid := range []int64{cmd.OperatorUserID, cmd.PeerUserID} {
			if err := s.addUserMember(txCtx, convID, uid, conversation.RoleMember); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		loggerx.WithError(ctx, s.logger, "create dm failed", err, slog.Int64("workspaceId", cmd.WorkspaceID))
		return nil, err
	}
	s.logger.InfoContext(ctx, "dm created", slog.Int64("conversationId", convID), slog.Int64("workspaceId", cmd.WorkspaceID))
	return toConversationResult(conv), nil
}

func (s *ConversationService) createGroupOrChannel(ctx context.Context, cmd messagingcmd.CreateConversationCommand) (*messagingresult.ConversationResult, error) {
	if err := s.checker.Check(ctx, authz.Request{
		WorkspaceID: cmd.WorkspaceID,
		Subject:     authz.Subject{UserID: cmd.OperatorUserID},
		Action:      actionChannelCreate,
	}); err != nil {
		return nil, err
	}

	convID, err := s.idGenerator.NextID(ctx)
	if err != nil {
		return nil, wrapInfrastructureError("生成会话 ID 失败", err)
	}
	var conv *conversation.Conversation
	if conversation.Type(cmd.Type) == conversation.TypeGroup {
		conv, err = conversation.NewGroup(convID, cmd.WorkspaceID, cmd.Name, cmd.OperatorUserID)
	} else {
		conv, err = conversation.NewChannel(convID, cmd.WorkspaceID, cmd.Name, conversation.Visibility(cmd.Visibility), cmd.OperatorUserID)
	}
	if err != nil {
		return nil, err
	}

	err = s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.conversations.Create(txCtx, conv); err != nil {
			return wrapInfrastructureError("保存会话失败", err)
		}
		return s.addUserMember(txCtx, convID, cmd.OperatorUserID, conversation.RoleOwner)
	})
	if err != nil {
		loggerx.WithError(ctx, s.logger, "create conversation failed", err, slog.Int64("workspaceId", cmd.WorkspaceID))
		return nil, err
	}
	s.logger.InfoContext(ctx, "conversation created", slog.Int64("conversationId", convID), slog.Int64("workspaceId", cmd.WorkspaceID))
	return toConversationResult(conv), nil
}

// addUserMember 在事务内创建一个用户成员记录。
func (s *ConversationService) addUserMember(ctx context.Context, convID, userID int64, role conversation.Role) error {
	memID, err := s.idGenerator.NextID(ctx)
	if err != nil {
		return wrapInfrastructureError("生成成员 ID 失败", err)
	}
	mem := conversation.NewMember(memID, convID, conversation.MemberUser, userID, role)
	if err := s.conversations.AddMember(ctx, mem); err != nil {
		return wrapInfrastructureError("加入会话成员失败", err)
	}
	return nil
}

// AddMember 向群聊/频道加入成员；需操作者为会话 owner/admin。
func (s *ConversationService) AddMember(ctx context.Context, cmd messagingcmd.AddMemberCommand) (*messagingresult.ConversationMemberResult, error) {
	if err := assertWorkspaceScope(ctx, cmd.WorkspaceID); err != nil {
		return nil, err
	}
	conv, err := s.conversations.FindByID(ctx, cmd.ConversationID)
	if err != nil {
		if stderrors.Is(err, conversation.ErrNotFound) {
			return nil, apperr.NotFound("会话不存在")
		}
		return nil, wrapInfrastructureError("查询会话失败", err)
	}
	if conv.Type() == conversation.TypeDM {
		return nil, apperr.InvalidParam("单聊不支持加人")
	}
	if err := s.requireConvManager(ctx, cmd.ConversationID, cmd.OperatorUserID); err != nil {
		return nil, err
	}

	memberType := conversation.MemberType(cmd.MemberType)
	if memberType != conversation.MemberUser && memberType != conversation.MemberBot {
		return nil, apperr.InvalidParam("成员类型非法")
	}
	if existing, err := s.conversations.FindMember(ctx, cmd.ConversationID, memberType, cmd.MemberID); err == nil {
		if existing.Status() == conversation.MemberActive {
			return nil, apperr.Conflict("已是会话成员")
		}
	} else if !stderrors.Is(err, conversation.ErrMemberNotFound) {
		return nil, wrapInfrastructureError("查询会话成员失败", err)
	}

	role := conversation.Role(cmd.Role)
	if role != conversation.RoleAdmin && role != conversation.RoleMember {
		role = conversation.RoleMember
	}

	var mem *conversation.Member
	err = s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		memID, err := s.idGenerator.NextID(txCtx)
		if err != nil {
			return wrapInfrastructureError("生成成员 ID 失败", err)
		}
		m := conversation.NewMember(memID, cmd.ConversationID, memberType, cmd.MemberID, role)
		if err := s.conversations.AddMember(txCtx, m); err != nil {
			return wrapInfrastructureError("加入会话成员失败", err)
		}
		conv.IncrMemberCount(1)
		if err := s.conversations.Save(txCtx, conv); err != nil {
			return wrapInfrastructureError("更新会话失败", err)
		}
		mem = m
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "conversation member added", slog.Int64("conversationId", cmd.ConversationID), slog.Int64("memberId", cmd.MemberID))
	return toConversationMemberResult(mem), nil
}

// RemoveMember 从群聊/频道移除成员；操作者需为 owner/admin 或移除自己。
func (s *ConversationService) RemoveMember(ctx context.Context, cmd messagingcmd.RemoveMemberCommand) error {
	if err := assertWorkspaceScope(ctx, cmd.WorkspaceID); err != nil {
		return err
	}
	conv, err := s.conversations.FindByID(ctx, cmd.ConversationID)
	if err != nil {
		if stderrors.Is(err, conversation.ErrNotFound) {
			return apperr.NotFound("会话不存在")
		}
		return wrapInfrastructureError("查询会话失败", err)
	}
	if conv.Type() == conversation.TypeDM {
		return apperr.InvalidParam("单聊不支持移除成员")
	}

	memberType := conversation.MemberType(cmd.MemberType)
	removingSelf := memberType == conversation.MemberUser && cmd.MemberID == cmd.OperatorUserID
	if !removingSelf {
		if err := s.requireConvManager(ctx, cmd.ConversationID, cmd.OperatorUserID); err != nil {
			return err
		}
	}

	target, err := s.conversations.FindMember(ctx, cmd.ConversationID, memberType, cmd.MemberID)
	if err != nil {
		if stderrors.Is(err, conversation.ErrMemberNotFound) {
			return apperr.NotFound("成员不存在")
		}
		return wrapInfrastructureError("查询会话成员失败", err)
	}
	if !target.Leave() {
		return nil
	}

	err = s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.conversations.SaveMember(txCtx, target); err != nil {
			return wrapInfrastructureError("更新会话成员失败", err)
		}
		conv.IncrMemberCount(-1)
		if err := s.conversations.Save(txCtx, conv); err != nil {
			return wrapInfrastructureError("更新会话失败", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "conversation member removed", slog.Int64("conversationId", cmd.ConversationID), slog.Int64("memberId", cmd.MemberID))
	return nil
}

// requireConvManager 校验操作者是会话的 owner/admin。
func (s *ConversationService) requireConvManager(ctx context.Context, convID, userID int64) error {
	mem, err := s.requireActiveMembership(ctx, convID, userID)
	if err != nil {
		return err
	}
	if mem.Role() != conversation.RoleOwner && mem.Role() != conversation.RoleAdmin {
		return apperr.Forbidden("需要会话管理员权限")
	}
	return nil
}

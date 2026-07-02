//go:build integration

package integration_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"entgo.io/ent/dialect/sql/schema"
	"github.com/stretchr/testify/suite"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/infra/localid"
	messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"
	messagingquery "github.com/maguowei/gotobeta/internal/modules/messaging/application/query"
	messagingsvc "github.com/maguowei/gotobeta/internal/modules/messaging/application/service"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/conversation"
	messagingpersist "github.com/maguowei/gotobeta/internal/modules/messaging/infra/persistence"
	"github.com/maguowei/gotobeta/internal/modules/messaging/infra/seqalloc"
	workspacecmd "github.com/maguowei/gotobeta/internal/modules/workspace/application/command"
	workspacesvc "github.com/maguowei/gotobeta/internal/modules/workspace/application/service"
	workspaceauthz "github.com/maguowei/gotobeta/internal/modules/workspace/infra/authz"
	workspacepersist "github.com/maguowei/gotobeta/internal/modules/workspace/infra/persistence"
	"github.com/maguowei/gotobeta/internal/modules/workspace/infra/seed"
	"github.com/maguowei/gotobeta/internal/pkg/event"
	"github.com/maguowei/gotobeta/internal/testutil"
)

// reactionNoopPublisher 吞掉事件发布：集成测试只验证持久化与幂等，不涉及实时扇出。
type reactionNoopPublisher struct{}

func (reactionNoopPublisher) Publish(context.Context, ...event.Event) error { return nil }

// MessagingReactionSuite 验证表情回应全链路：添加、幂等去重、列举、取消。
type MessagingReactionSuite struct {
	suite.Suite
	mysql   *testutil.MySQLContainer
	client  *ent.Client
	wsSvc   *workspacesvc.WorkspaceService
	convSvc *messagingsvc.ConversationService
	msgSvc  *messagingsvc.MessageService
}

func (s *MessagingReactionSuite) SetupSuite() {
	ctx := context.Background()
	s.mysql = testutil.StartMySQL(ctx, s.T())

	client, sqlDB, err := entdb.NewEntClient(&config.DatabaseConfig{
		Driver: "mysql",
		DSN:    s.mysql.DSN,
	})
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		s.Require().NoError(client.Close())
		s.Require().NoError(sqlDB.Close())
	})

	s.Require().NoError(client.Schema.Create(ctx, schema.WithForeignKeys(false)))
	s.client = client

	logger := slog.New(slog.DiscardHandler)
	idGen := localid.New()
	wsRepo := workspacepersist.NewWorkspaceRepository(client, logger)
	memRepo := workspacepersist.NewMembershipRepository(client, logger)
	rbacRepo := workspacepersist.NewRBACRepository(client, logger, idGen)
	aclRepo := workspacepersist.NewACLRepository(client, logger)
	checker := workspaceauthz.NewChecker(rbacRepo, aclRepo)
	txRunner := entdb.NewEntTxRunner(client)

	// 平台模板必须先 seed，CreateWorkspace 才能复制角色与权限（含 message.react）。
	s.Require().NoError(seed.SeedPlatformTemplates(ctx, rbacRepo, idGen))

	s.wsSvc = workspacesvc.NewWorkspaceService(wsRepo, memRepo, rbacRepo, checker, idGen, txRunner, logger)
	convRepo := messagingpersist.NewConversationRepository(client, logger)
	s.convSvc = messagingsvc.NewConversationService(convRepo, checker, idGen, txRunner, logger)
	s.msgSvc = messagingsvc.NewMessageService(
		messagingpersist.NewMessageRepository(client, logger),
		convRepo,
		messagingpersist.NewReactionRepository(client, logger),
		messagingpersist.NewMessageChangeRepository(client, logger),
		seqalloc.NewDBAllocator(client),
		checker, reactionNoopPublisher{}, idGen, txRunner,
		time.Minute, 50, logger, nil,
	)
}

// TestReactionLifecycle 覆盖添加→幂等→列举→取消全链路及唯一约束去重。
func (s *MessagingReactionSuite) TestReactionLifecycle() {
	ctx := context.Background()
	const ownerID int64 = 9001

	ws, err := s.wsSvc.CreateWorkspace(ctx, workspacecmd.CreateWorkspaceCommand{
		Slug:        "react-team",
		Name:        "React Team",
		OwnerUserID: ownerID,
	})
	s.Require().NoError(err)

	// 创建群聊，owner 自动成为成员。
	conv, err := s.convSvc.CreateConversation(ctx, messagingcmd.CreateConversationCommand{
		WorkspaceID:    ws.ID,
		OperatorUserID: ownerID,
		Type:           int8(conversation.TypeGroup),
		Name:           "general",
	})
	s.Require().NoError(err)

	// 发送一条消息作为回应目标。
	msg, err := s.msgSvc.SendMessage(ctx, messagingcmd.SendMessageCommand{
		WorkspaceID:    ws.ID,
		ConversationID: conv.ID,
		SenderUserID:   ownerID,
		ClientMsgID:    "c-react-1",
		ContentType:    1,
		Content:        map[string]any{"text": "hi"},
	})
	s.Require().NoError(err)

	addCmd := messagingcmd.AddReactionCommand{
		WorkspaceID:    ws.ID,
		ConversationID: conv.ID,
		MessageID:      msg.MessageID,
		OperatorUserID: ownerID,
		Emoji:          "👍",
	}

	// 添加表情回应。
	s.Require().NoError(s.msgSvc.AddReaction(ctx, addCmd))

	listQuery := messagingquery.ListReactionsQuery{
		WorkspaceID:    ws.ID,
		ConversationID: conv.ID,
		MessageID:      msg.MessageID,
		OperatorUserID: ownerID,
	}
	list, err := s.msgSvc.ListReactions(ctx, listQuery)
	s.Require().NoError(err)
	s.Require().Len(list, 1)
	s.Equal("👍", list[0].Emoji)
	s.Equal(ownerID, list[0].UserID)

	// 幂等：重复添加同一 emoji 不应新增（唯一约束去重）。
	s.Require().NoError(s.msgSvc.AddReaction(ctx, addCmd))
	list, err = s.msgSvc.ListReactions(ctx, listQuery)
	s.Require().NoError(err)
	s.Require().Len(list, 1)

	// 取消回应。
	s.Require().NoError(s.msgSvc.RemoveReaction(ctx, messagingcmd.RemoveReactionCommand{
		WorkspaceID:    ws.ID,
		ConversationID: conv.ID,
		MessageID:      msg.MessageID,
		OperatorUserID: ownerID,
		Emoji:          "👍",
	}))
	list, err = s.msgSvc.ListReactions(ctx, listQuery)
	s.Require().NoError(err)
	s.Require().Empty(list)
}

func TestMessagingReactionSuite(t *testing.T) {
	suite.Run(t, new(MessagingReactionSuite))
}

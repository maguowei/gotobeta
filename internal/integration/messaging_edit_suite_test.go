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

// editNoopPublisher 吞掉事件发布：集成测试只验证持久化，不涉及实时扇出。
type editNoopPublisher struct{}

func (editNoopPublisher) Publish(context.Context, ...event.Event) error { return nil }

// MessagingEditSuite 验证消息编辑全链路：原地更新内容、记录 editedAt、仅本人可编辑。
type MessagingEditSuite struct {
	suite.Suite
	mysql   *testutil.MySQLContainer
	client  *ent.Client
	wsSvc   *workspacesvc.WorkspaceService
	convSvc *messagingsvc.ConversationService
	msgSvc  *messagingsvc.MessageService
}

func (s *MessagingEditSuite) SetupSuite() {
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
		checker, editNoopPublisher{}, idGen, txRunner,
		time.Minute, 50, logger, nil,
	)
}

// TestEditLifecycle 覆盖发送→编辑→拉取看到新内容与 editedAt→非本人编辑被拒。
func (s *MessagingEditSuite) TestEditLifecycle() {
	ctx := context.Background()
	const ownerID int64 = 9101

	ws, err := s.wsSvc.CreateWorkspace(ctx, workspacecmd.CreateWorkspaceCommand{
		Slug:        "edit-team",
		Name:        "Edit Team",
		OwnerUserID: ownerID,
	})
	s.Require().NoError(err)

	conv, err := s.convSvc.CreateConversation(ctx, messagingcmd.CreateConversationCommand{
		WorkspaceID:    ws.ID,
		OperatorUserID: ownerID,
		Type:           int8(conversation.TypeGroup),
		Name:           "general",
	})
	s.Require().NoError(err)

	msg, err := s.msgSvc.SendMessage(ctx, messagingcmd.SendMessageCommand{
		WorkspaceID:    ws.ID,
		ConversationID: conv.ID,
		SenderUserID:   ownerID,
		ClientMsgID:    "c-edit-1",
		ContentType:    1,
		Content:        map[string]any{"text": "原始内容"},
	})
	s.Require().NoError(err)

	// 本人编辑：原地更新内容并返回 editedAt。
	edited, err := s.msgSvc.EditMessage(ctx, messagingcmd.EditMessageCommand{
		WorkspaceID:    ws.ID,
		ConversationID: conv.ID,
		OperatorUserID: ownerID,
		MessageID:      msg.MessageID,
		Content:        map[string]any{"text": "编辑后内容"},
	})
	s.Require().NoError(err)
	s.Equal("编辑后内容", edited.Content["text"])
	s.Require().NotNil(edited.EditedAt)

	// 编辑不占新 seq：拉取仍只有 1 条，且内容已更新、editedAt 持久化。
	items, err := s.msgSvc.PullMessages(ctx, messagingquery.PullMessagesQuery{
		WorkspaceID:    ws.ID,
		OperatorUserID: ownerID,
		ConversationID: conv.ID,
		AfterSeq:       0,
		Limit:          50,
	})
	s.Require().NoError(err)
	s.Require().Len(items, 1)
	s.Equal("编辑后内容", items[0].Content["text"])
	s.Require().NotNil(items[0].EditedAt)
	s.Equal(int64(1), items[0].Seq)

	// 非本人（另一活跃成员）编辑应被拒绝。
	const otherID int64 = 9102
	_, err = s.convSvc.AddMember(ctx, messagingcmd.AddMemberCommand{
		WorkspaceID:    ws.ID,
		OperatorUserID: ownerID,
		ConversationID: conv.ID,
		MemberType:     int8(conversation.MemberUser),
		MemberID:       otherID,
		Role:           int8(conversation.RoleMember),
	})
	s.Require().NoError(err)
	_, err = s.msgSvc.EditMessage(ctx, messagingcmd.EditMessageCommand{
		WorkspaceID:    ws.ID,
		ConversationID: conv.ID,
		OperatorUserID: otherID,
		MessageID:      msg.MessageID,
		Content:        map[string]any{"text": "篡改"},
	})
	s.Require().Error(err)
}

func TestMessagingEditSuite(t *testing.T) {
	suite.Run(t, new(MessagingEditSuite))
}

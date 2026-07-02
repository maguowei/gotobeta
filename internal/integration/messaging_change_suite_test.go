//go:build integration

package integration_test

import (
	"context"
	"log/slog"
	"strconv"
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
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/messagechange"
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

// changeNoopPublisher 吞掉事件发布：集成测试只验证持久化，不涉及实时扇出。
type changeNoopPublisher struct{}

func (changeNoopPublisher) Publish(context.Context, ...event.Event) error { return nil }

// MessagingChangeSuite 验证会话变更增量同步全链路：发消息/编辑/加 reaction/撤回后 ListChanges 追平与续拉。
type MessagingChangeSuite struct {
	suite.Suite
	mysql   *testutil.MySQLContainer
	client  *ent.Client
	wsSvc   *workspacesvc.WorkspaceService
	convSvc *messagingsvc.ConversationService
	msgSvc  *messagingsvc.MessageService
}

func (s *MessagingChangeSuite) SetupSuite() {
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
		checker, changeNoopPublisher{}, idGen, txRunner,
		time.Minute, 50, logger, nil,
	)
}

// TestChangeStreamCatchUp 覆盖发消息→编辑→加 reaction 后一次追平，以及从中间 changeSeq 续拉。
func (s *MessagingChangeSuite) TestChangeStreamCatchUp() {
	ctx := context.Background()
	const ownerID int64 = 9201

	ws, err := s.wsSvc.CreateWorkspace(ctx, workspacecmd.CreateWorkspaceCommand{Slug: "chg-team", Name: "Chg", OwnerUserID: ownerID})
	s.Require().NoError(err)
	conv, err := s.convSvc.CreateConversation(ctx, messagingcmd.CreateConversationCommand{
		WorkspaceID: ws.ID, OperatorUserID: ownerID, Type: int8(conversation.TypeGroup), Name: "g",
	})
	s.Require().NoError(err)

	// 发消息 → 编辑 → 加 reaction。
	msg, err := s.msgSvc.SendMessage(ctx, messagingcmd.SendMessageCommand{
		WorkspaceID: ws.ID, ConversationID: conv.ID, SenderUserID: ownerID,
		ClientMsgID: "c1", ContentType: 1, Content: map[string]any{"text": "原始"},
	})
	s.Require().NoError(err)
	_, err = s.msgSvc.EditMessage(ctx, messagingcmd.EditMessageCommand{
		WorkspaceID: ws.ID, ConversationID: conv.ID, OperatorUserID: ownerID,
		MessageID: msg.MessageID, Content: map[string]any{"text": "编辑后"},
	})
	s.Require().NoError(err)
	err = s.msgSvc.AddReaction(ctx, messagingcmd.AddReactionCommand{
		WorkspaceID: ws.ID, ConversationID: conv.ID, MessageID: msg.MessageID, OperatorUserID: ownerID, Emoji: "👍",
	})
	s.Require().NoError(err)

	// 一次追平：afterChangeSeq=0 应拉回 created + edited + reaction_add。
	page, err := s.msgSvc.ListChanges(ctx, messagingquery.ListChangesQuery{
		WorkspaceID: ws.ID, OperatorUserID: ownerID, ConversationID: conv.ID, AfterChangeSeq: 0, Limit: 50,
	})
	s.Require().NoError(err)
	s.Require().Len(page.Changes, 3)
	// change_seq 严格递增无间隙。
	s.Equal(int8(1), page.Changes[0].ChangeType) // created
	s.Equal(int8(2), page.Changes[1].ChangeType) // edited
	s.Equal(int8(3), page.Changes[2].ChangeType) // reaction_add
	s.Greater(page.Changes[1].ChangeSeq, page.Changes[0].ChangeSeq)
	s.Greater(page.Changes[2].ChangeSeq, page.Changes[1].ChangeSeq)
	// edited payload 带新内容。
	s.NotNil(page.Changes[1].Payload["content"])
	// reaction_add payload 的 userId 以字符串承载（避免大整数丢精度）。
	s.Equal(strconv.FormatInt(ownerID, 10), page.Changes[2].Payload["userId"])

	// 增量续拉：从第 1 条之后拉，应剩 2 条。
	page2, err := s.msgSvc.ListChanges(ctx, messagingquery.ListChangesQuery{
		WorkspaceID: ws.ID, OperatorUserID: ownerID, ConversationID: conv.ID,
		AfterChangeSeq: page.Changes[0].ChangeSeq, Limit: 50,
	})
	s.Require().NoError(err)
	s.Require().Len(page2.Changes, 2)
}

// TestRecallAppearsAsCreatedInStream 回归 R1：撤回在变更流里表现为 changeType=1(created)，payload 携带 recalledMsgId。
func (s *MessagingChangeSuite) TestRecallAppearsAsCreatedInStream() {
	ctx := context.Background()
	const ownerID int64 = 9202
	ws, err := s.wsSvc.CreateWorkspace(ctx, workspacecmd.CreateWorkspaceCommand{Slug: "chg-team2", Name: "Chg2", OwnerUserID: ownerID})
	s.Require().NoError(err)
	conv, err := s.convSvc.CreateConversation(ctx, messagingcmd.CreateConversationCommand{
		WorkspaceID: ws.ID, OperatorUserID: ownerID, Type: int8(conversation.TypeGroup), Name: "g",
	})
	s.Require().NoError(err)
	msg, err := s.msgSvc.SendMessage(ctx, messagingcmd.SendMessageCommand{
		WorkspaceID: ws.ID, ConversationID: conv.ID, SenderUserID: ownerID,
		ClientMsgID: "c1", ContentType: 1, Content: map[string]any{"text": "hi"},
	})
	s.Require().NoError(err)
	s.Require().NoError(s.msgSvc.RecallMessage(ctx, messagingcmd.RecallMessageCommand{
		WorkspaceID: ws.ID, ConversationID: conv.ID, OperatorUserID: ownerID, MessageID: msg.MessageID,
	}))
	page, err := s.msgSvc.ListChanges(ctx, messagingquery.ListChangesQuery{
		WorkspaceID: ws.ID, OperatorUserID: ownerID, ConversationID: conv.ID, AfterChangeSeq: 0, Limit: 50,
	})
	s.Require().NoError(err)
	// 原消息 created + 撤回系统条目 created，共 2 条，末条 payload 含 recalledMsgId。
	s.Require().Len(page.Changes, 2)
	last := page.Changes[len(page.Changes)-1]
	s.Equal(int8(1), last.ChangeType)
	// recalledMsgId 以字符串承载（大整数经 JSON number 会丢精度），断言等于被撤回消息 ID 的十进制串。
	s.Require().NotNil(last.Payload["recalledMsgId"])
	s.Equal(strconv.FormatInt(msg.MessageID, 10), last.Payload["recalledMsgId"])
}

// TestAppendDuplicateChangeSeqRejected 验证 (conversation_id, change_seq) 唯一索引兜底转领域错误。
func (s *MessagingChangeSuite) TestAppendDuplicateChangeSeqRejected() {
	ctx := context.Background()
	repo := messagingpersist.NewMessageChangeRepository(s.client, slog.New(slog.DiscardHandler))
	const convID, seq int64 = 77001, 1

	c1, err := messagechange.New(90001, convID, seq, messagechange.ChangeCreated, 8001, 9, map[string]any{})
	s.Require().NoError(err)
	s.Require().NoError(repo.Append(ctx, c1))

	// 同一 (conversation_id, change_seq) 再插一条 → 撞唯一索引，应转 ErrDuplicateChangeSeq。
	c2, err := messagechange.New(90002, convID, seq, messagechange.ChangeCreated, 8002, 9, map[string]any{})
	s.Require().NoError(err)
	s.Require().ErrorIs(repo.Append(ctx, c2), messagechange.ErrDuplicateChangeSeq)
}

func TestMessagingChangeSuite(t *testing.T) {
	suite.Run(t, new(MessagingChangeSuite))
}

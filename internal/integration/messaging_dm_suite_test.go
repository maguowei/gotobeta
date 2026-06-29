//go:build integration

package integration_test

import (
	"context"
	"log/slog"
	"testing"

	"entgo.io/ent/dialect/sql/schema"
	"github.com/stretchr/testify/suite"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/infra/localid"
	messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"
	messagingsvc "github.com/maguowei/gotobeta/internal/modules/messaging/application/service"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/conversation"
	messagingpersist "github.com/maguowei/gotobeta/internal/modules/messaging/infra/persistence"
	workspacecmd "github.com/maguowei/gotobeta/internal/modules/workspace/application/command"
	workspacesvc "github.com/maguowei/gotobeta/internal/modules/workspace/application/service"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
	workspaceauthz "github.com/maguowei/gotobeta/internal/modules/workspace/infra/authz"
	workspacepersist "github.com/maguowei/gotobeta/internal/modules/workspace/infra/persistence"
	"github.com/maguowei/gotobeta/internal/modules/workspace/infra/seed"
	"github.com/maguowei/gotobeta/internal/testutil"
)

// MessagingDMSuite 验证单聊 dm_key 对称去重：两个方向各发起一次单聊命中同一会话。
type MessagingDMSuite struct {
	suite.Suite
	mysql   *testutil.MySQLContainer
	client  *ent.Client
	wsSvc   *workspacesvc.WorkspaceService
	convSvc *messagingsvc.ConversationService
}

func (s *MessagingDMSuite) SetupSuite() {
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

	// 平台模板必须先 seed，CreateWorkspace 才能复制角色与权限。
	s.Require().NoError(seed.SeedPlatformTemplates(ctx, rbacRepo, idGen))

	s.wsSvc = workspacesvc.NewWorkspaceService(wsRepo, memRepo, rbacRepo, checker, idGen, txRunner, logger)
	// messaging 会话服务复用同一 checker：单聊发起需 conversation.read 动作。
	s.convSvc = messagingsvc.NewConversationService(
		messagingpersist.NewConversationRepository(client, logger),
		checker, idGen, txRunner, logger,
	)
}

// TestDMKeyDedupBothDirections 两个用户互相发起单聊（双向），断言命中同一会话。
func (s *MessagingDMSuite) TestDMKeyDedupBothDirections() {
	ctx := context.Background()
	const ownerID int64 = 7001
	const peerID int64 = 7002

	ws, err := s.wsSvc.CreateWorkspace(ctx, workspacecmd.CreateWorkspaceCommand{
		Slug:        "dm-team",
		Name:        "DM Team",
		OwnerUserID: ownerID,
	})
	s.Require().NoError(err)

	// peer 加入工作区并赋 member 角色（拥有 conversation.read），使其也能发起单聊。
	_, err = s.wsSvc.InviteMember(ctx, workspacecmd.InviteMemberCommand{
		WorkspaceID:    ws.ID,
		OperatorUserID: ownerID,
		TargetUserID:   peerID,
		RoleCode:       rbac.RoleMember,
	})
	s.Require().NoError(err)

	// 方向一：owner -> peer，创建单聊。
	conv1, err := s.convSvc.CreateConversation(ctx, messagingcmd.CreateConversationCommand{
		WorkspaceID:    ws.ID,
		OperatorUserID: ownerID,
		Type:           int8(conversation.TypeDM),
		PeerUserID:     peerID,
	})
	s.Require().NoError(err)

	// 方向二：peer -> owner，应命中同一会话（dm_key 对称去重）。
	conv2, err := s.convSvc.CreateConversation(ctx, messagingcmd.CreateConversationCommand{
		WorkspaceID:    ws.ID,
		OperatorUserID: peerID,
		Type:           int8(conversation.TypeDM),
		PeerUserID:     ownerID,
	})
	s.Require().NoError(err)

	// 断言：两个方向返回同一会话 ID 与同一 dm_key。
	s.Equal(conv1.ID, conv2.ID)
	s.Require().NotNil(conv1.DMKey)
	s.Require().NotNil(conv2.DMKey)
	s.Equal(*conv1.DMKey, *conv2.DMKey)
}

func TestMessagingDMSuite(t *testing.T) {
	suite.Run(t, new(MessagingDMSuite))
}

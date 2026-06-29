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
	workspacecmd "github.com/maguowei/gotobeta/internal/modules/workspace/application/command"
	workspacesvc "github.com/maguowei/gotobeta/internal/modules/workspace/application/service"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
	workspaceauthz "github.com/maguowei/gotobeta/internal/modules/workspace/infra/authz"
	workspacepersist "github.com/maguowei/gotobeta/internal/modules/workspace/infra/persistence"
	"github.com/maguowei/gotobeta/internal/modules/workspace/infra/seed"
	"github.com/maguowei/gotobeta/internal/testutil"
)

// RBACSuite 验证动态 RBAC：建工作区复制平台模板、AssignRole 后 ResolveUserActions 返回角色动作集合。
type RBACSuite struct {
	suite.Suite
	mysql    *testutil.MySQLContainer
	client   *ent.Client
	svc      *workspacesvc.WorkspaceService
	rbacRepo *workspacepersist.RBACRepository
}

func (s *RBACSuite) SetupSuite() {
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
	wsRepo := workspacepersist.NewWorkspaceRepository(client, logger)
	memRepo := workspacepersist.NewMembershipRepository(client, logger)
	rbacRepo := workspacepersist.NewRBACRepository(client, logger)
	aclRepo := workspacepersist.NewACLRepository(client, logger)
	checker := workspaceauthz.NewChecker(rbacRepo, aclRepo)
	idGen := localid.New()
	txRunner := entdb.NewEntTxRunner(client)

	// 平台模板（workspace_id=0）必须先 seed，CreateWorkspace 才能从模板复制角色并绑定权限。
	s.Require().NoError(seed.SeedPlatformTemplates(ctx, rbacRepo, idGen))

	s.svc = workspacesvc.NewWorkspaceService(wsRepo, memRepo, rbacRepo, checker, idGen, txRunner, logger)
	s.rbacRepo = rbacRepo
}

// TestResolveUserActionsAfterAssignRole 建工作区（复制平台角色+权限），
// 给某用户 AssignRole(member)，断言 ResolveUserActions 返回 member 角色对应的动作集合。
func (s *RBACSuite) TestResolveUserActionsAfterAssignRole() {
	ctx := context.Background()
	const ownerID int64 = 9001
	const memberID int64 = 9002

	ws, err := s.svc.CreateWorkspace(ctx, workspacecmd.CreateWorkspaceCommand{
		Slug:        "rbac-team",
		Name:        "RBAC Team",
		OwnerUserID: ownerID,
	})
	s.Require().NoError(err)

	// owner 拥有 role.manage，可给目标用户分配 member 角色。
	err = s.svc.AssignRole(ctx, workspacecmd.AssignRoleCommand{
		WorkspaceID:    ws.ID,
		OperatorUserID: ownerID,
		TargetUserID:   memberID,
		RoleCode:       rbac.RoleMember,
	})
	s.Require().NoError(err)

	actions, err := s.rbacRepo.ResolveUserActions(ctx, ws.ID, memberID)
	s.Require().NoError(err)

	// 断言：动作集合恰好等于 member 角色默认绑定的权限编码集合。
	expectedCodes := rbac.DefaultRolePermissions()[rbac.RoleMember]
	expected := make(map[string]struct{}, len(expectedCodes))
	for _, code := range expectedCodes {
		expected[code] = struct{}{}
	}
	s.Equal(expected, actions)
}

func TestRBACSuite(t *testing.T) {
	suite.Run(t, new(RBACSuite))
}

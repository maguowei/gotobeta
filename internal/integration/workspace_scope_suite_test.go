//go:build integration

package integration_test

import (
	"context"
	"testing"

	"entgo.io/ent/dialect/sql/schema"
	"github.com/stretchr/testify/suite"

	"github.com/maguowei/gotobeta/internal/ent"
	entrole "github.com/maguowei/gotobeta/internal/ent/rbacrole"
	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/infra/localid"
	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
	"github.com/maguowei/gotobeta/internal/testutil"
)

// WorkspaceScopeSuite 验证 entdb.WorkspaceScopeInterceptor 在 repo 层兜底工作区隔离：
// context 携带 workspace_id 时查询只返回该工作区数据，显式逃逸时返回全部。
type WorkspaceScopeSuite struct {
	suite.Suite
	mysql  *testutil.MySQLContainer
	client *ent.Client
	idgen  *localid.Generator
}

func (s *WorkspaceScopeSuite) SetupSuite() {
	ctx := context.Background()
	s.mysql = testutil.StartMySQL(ctx, s.T())

	client, sqlDB, err := entdb.NewEntClient(&config.DatabaseConfig{Driver: "mysql", DSN: s.mysql.DSN})
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		s.Require().NoError(client.Close())
		s.Require().NoError(sqlDB.Close())
	})
	s.Require().NoError(client.Schema.Create(ctx, schema.WithForeignKeys(false)))
	s.client = client
	s.idgen = localid.New()
}

func (s *WorkspaceScopeSuite) createRole(ctx context.Context, wsID int64, code string) {
	bizID, err := s.idgen.NextID(ctx)
	s.Require().NoError(err)
	_, err = s.client.RbacRole.Create().
		SetBizID(bizID).
		SetWorkspaceID(wsID).
		SetCode(code).
		SetName(code).
		SetRoleType(2).
		SetStatus(1).
		Save(ctx)
	s.Require().NoError(err)
}

func (s *WorkspaceScopeSuite) TestQueryScopedToWorkspace() {
	base := context.Background()
	// 用逃逸 context 写入两个工作区的数据，避免拦截器影响写前的存在性查询。
	seed := entdb.WithoutWorkspaceScope(base)
	s.createRole(seed, 1001, "ws1-role")
	s.createRole(seed, 1002, "ws2-role")

	// 携带 ws1 的 context：只应看到 ws1 的角色。
	ctx1 := requestctx.WithWorkspaceID(base, 1001)
	roles, err := s.client.RbacRole.Query().All(ctx1)
	s.Require().NoError(err)
	for _, r := range roles {
		s.Equalf(int64(1001), r.WorkspaceID, "拦截器泄漏了非 ws1 数据: %s", r.Code)
	}
	s.NotEmpty(roles)

	// 逃逸 context：应看到全部工作区数据。
	all, err := s.client.RbacRole.Query().Where(entrole.WorkspaceIDIn(1001, 1002)).All(entdb.WithoutWorkspaceScope(base))
	s.Require().NoError(err)
	s.GreaterOrEqual(len(all), 2)
}

func TestWorkspaceScopeSuite(t *testing.T) {
	suite.Run(t, new(WorkspaceScopeSuite))
}

//go:build integration

package integration_test

import (
	"context"
	"testing"

	"entgo.io/ent/dialect/sql/schema"
	"github.com/stretchr/testify/suite"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/testutil"
)

type ExampleSuite struct {
	suite.Suite
	mysql  *testutil.MySQLContainer
	client *ent.Client
}

func (s *ExampleSuite) SetupSuite() {
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
}

func (s *ExampleSuite) TestDatabaseIsWritable() {
	ctx := context.Background()
	item, err := s.client.AppSetting.Create().
		SetKey("integration-smoke").
		SetValue("ok").
		Save(ctx)
	s.Require().NoError(err)
	s.Equal("integration-smoke", item.Key)
}

func TestExampleSuite(t *testing.T) {
	suite.Run(t, new(ExampleSuite))
}

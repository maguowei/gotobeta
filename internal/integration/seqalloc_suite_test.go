//go:build integration

package integration_test

import (
	"context"
	"slices"
	"testing"

	"entgo.io/ent/dialect/sql/schema"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/errgroup"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/messaging/infra/seqalloc"
	"github.com/maguowei/gotobeta/internal/testutil"
)

// SeqAllocSuite 验证同一会话内 seq 分配在并发下连续、唯一、无空洞。
type SeqAllocSuite struct {
	suite.Suite
	mysql    *testutil.MySQLContainer
	client   *ent.Client
	txRunner *entdb.EntTxRunner
}

func (s *SeqAllocSuite) SetupSuite() {
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
	s.txRunner = entdb.NewEntTxRunner(client)
}

// TestConcurrentNextIsContiguousAndUnique 对同一会话并发分配 N 次 seq，
// 断言得到 1..N 的连续且唯一集合（无重复、无空洞）。
func (s *SeqAllocSuite) TestConcurrentNextIsContiguousAndUnique() {
	ctx := context.Background()
	const convID int64 = 1001
	const n = 50

	// 预置一条会话行，last_seq 从 0 起；DBAllocator 通过行锁原子推进。
	_, err := s.client.Conversation.Create().
		SetBizID(convID).
		SetWorkspaceID(1).
		SetType(2).
		SetVisibility(2).
		SetName("seqalloc-conv").
		SetCreatorID(1).
		SetLastSeq(0).
		SetMemberCount(1).
		SetStatus(1).
		Save(ctx)
	s.Require().NoError(err)

	allocator := seqalloc.NewDBAllocator(s.client)

	results := make([]int64, n)
	g, gctx := errgroup.WithContext(ctx)
	for i := range n {
		g.Go(func() error {
			// 每次分配独立开事务：行锁持有至提交，使并发分配串行化。
			return s.txRunner.RunInTx(gctx, func(txCtx context.Context) error {
				seq, err := allocator.Next(txCtx, convID)
				if err != nil {
					return err
				}
				results[i] = seq
				return nil
			})
		})
	}
	s.Require().NoError(g.Wait())

	// 断言：集合恰好是 1..N，无重复、无空洞。
	sorted := slices.Clone(results)
	slices.Sort(sorted)
	expected := make([]int64, n)
	for i := range n {
		expected[i] = int64(i + 1)
	}
	s.Equal(expected, sorted)
}

func TestSeqAllocSuite(t *testing.T) {
	suite.Run(t, new(SeqAllocSuite))
}

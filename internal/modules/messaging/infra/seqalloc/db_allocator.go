// Package seqalloc 提供会话内序列号（seq）的分配实现。
package seqalloc

import (
	"context"
	"fmt"

	"github.com/maguowei/gotobeta/internal/ent"
	entconv "github.com/maguowei/gotobeta/internal/ent/conversation"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/conversation"
)

// DBAllocator 基于数据库行锁实现 seq 分配。
//
// 通过对 conversation 行执行原子自增 UPDATE（last_seq = last_seq + 1）获取行级写锁，
// 锁持有至事务提交，使并发分配天然串行化；随后回读得到新分配的 seq。
// 必须在事务（TxRunner.RunInTx）内调用。
type DBAllocator struct {
	client *ent.Client
}

// NewDBAllocator 创建分配器。
func NewDBAllocator(client *ent.Client) *DBAllocator {
	return &DBAllocator{client: client}
}

// Next 原子推进并返回会话 convID 的下一个 seq。
func (a *DBAllocator) Next(ctx context.Context, convID int64) (int64, error) {
	client := entdb.ClientFromCtx(ctx, a.client)
	affected, err := client.Conversation.Update().
		Where(entconv.BizID(convID)).
		AddLastSeq(1).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("seqalloc: 自增 last_seq: %w", err)
	}
	if affected == 0 {
		return 0, conversation.ErrNotFound
	}
	row, err := client.Conversation.Query().
		Where(entconv.BizID(convID)).
		Select(entconv.FieldLastSeq).
		Only(ctx)
	if err != nil {
		return 0, fmt.Errorf("seqalloc: 回读 last_seq: %w", err)
	}
	return row.LastSeq, nil
}

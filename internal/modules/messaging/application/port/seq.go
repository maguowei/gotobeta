// Package port 定义 messaging 模块对基础设施的出站端口。
package port

import "context"

// SeqAllocator 为会话分配单调递增的会话内序列号（seq）。
//
// 第一期由 DB 行锁实现（SELECT ... FOR UPDATE 锁 conversation 行后 ++last_seq），
// 必须在事务内调用；后续可替换为 Redis INCR / 独立 seqsvr。
type SeqAllocator interface {
	// Next 返回会话 convID 的下一个 seq，并持久化推进 last_seq。
	Next(ctx context.Context, convID int64) (int64, error)
}

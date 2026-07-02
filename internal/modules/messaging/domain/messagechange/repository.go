package messagechange

import "context"

// Repository 是变更流仓储接口。
type Repository interface {
	// Append 追加一条变更记录（在业务写事务内调用，保证原子）。
	Append(ctx context.Context, c *Change) error
	// ListAfter 返回会话内 change_seq > afterChangeSeq 的变更，按 change_seq 升序，最多 limit 条。
	ListAfter(ctx context.Context, conversationID, afterChangeSeq int64, limit int) ([]*Change, error)
}

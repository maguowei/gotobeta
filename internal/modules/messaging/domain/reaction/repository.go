package reaction

import "context"

// Repository 定义表情回应聚合的仓储接口。
type Repository interface {
	// Add 保存一条回应；命中唯一约束（同用户同消息同 emoji 已存在）返回 ErrAlreadyExists。
	Add(ctx context.Context, r *Reaction) error
	// Remove 删除指定回应，返回是否删除了记录（不存在时 false，幂等）。
	Remove(ctx context.Context, messageID, userID int64, emoji string) (bool, error)
	// ListByMessageID 返回消息的全部回应，按创建时间升序。
	ListByMessageID(ctx context.Context, messageID int64) ([]*Reaction, error)
}

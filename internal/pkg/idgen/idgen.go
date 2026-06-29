package idgen

import "context"

// Generator 提供 ID 生成能力（由基础设施层实现，注入应用层）。
type Generator interface {
	NextID(ctx context.Context) (int64, error)
}

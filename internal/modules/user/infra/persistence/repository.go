package persistence

import (
	"github.com/maguowei/gotobeta/internal/ent"
)

// 各聚合仓储接口都使用 Create/Consume 等通用方法名，无法由同一个类型同时实现，
// 因此按聚合拆分为独立的仓储类型，各自实现对应接口；本文件只保留它们共享的辅助函数。

// mapEntNotFound 把 Ent 的 not found 错误映射为对应聚合的哨兵错误。
func mapEntNotFound(err error, notFound error) error {
	if ent.IsNotFound(err) {
		return notFound
	}
	return err
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func valueString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

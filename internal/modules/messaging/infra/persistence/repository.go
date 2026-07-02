package persistence

import (
	"github.com/maguowei/gotobeta/internal/ent"
)

// mapEntNotFound 把 Ent 的 not found 错误映射为对应聚合的哨兵错误。
func mapEntNotFound(err error, notFound error) error {
	if ent.IsNotFound(err) {
		return notFound
	}
	return err
}

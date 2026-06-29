package result

import "time"

// TodoResult 是待办用例的应用层结果。
type TodoResult struct {
	ID        int64
	Title     string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

package todo

// Status 表示待办状态。
type Status string

const (
	StatusPending Status = "pending"
	StatusDone    Status = "done"
)

// IsValid 校验状态值是否合法。
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusDone:
		return true
	}
	return false
}

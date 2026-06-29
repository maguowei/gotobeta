package user

// Status 是用户账号状态。
type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
)

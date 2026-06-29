package workspace

// Status 表示工作区状态。
type Status int8

const (
	// StatusActive 正常。
	StatusActive Status = 1
	// StatusDisabled 停用。
	StatusDisabled Status = 2
)

// IsValid 校验状态值是否合法。
func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusDisabled:
		return true
	}
	return false
}

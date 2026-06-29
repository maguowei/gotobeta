package identity

import "errors"

var (
	// ErrNotFound 表示第三方身份不存在。
	ErrNotFound = errors.New("identity: not found")
)

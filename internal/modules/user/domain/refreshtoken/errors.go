package refreshtoken

import "errors"

var (
	// ErrNotFound 表示 refresh token 不存在或已失效。
	ErrNotFound = errors.New("refresh token: not found")
)

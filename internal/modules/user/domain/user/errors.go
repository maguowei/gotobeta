package user

import "errors"

var (
	// ErrNotFound 表示用户不存在。
	ErrNotFound = errors.New("user: not found")
)

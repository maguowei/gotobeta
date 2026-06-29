package attachment

import "errors"

var (
	// ErrNotFound 表示附件不存在。
	ErrNotFound = errors.New("attachment: not found")
)

package todo

import "errors"

var (
	// ErrNotFound 表示待办不存在。
	ErrNotFound = errors.New("todo: not found")
	// ErrConflict 表示待办已存在。
	ErrConflict = errors.New("todo: conflict")
)

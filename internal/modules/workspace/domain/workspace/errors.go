package workspace

import "errors"

var (
	// ErrNotFound 表示工作区不存在。
	ErrNotFound = errors.New("workspace: not found")
	// ErrSlugTaken 表示工作区标识已被占用。
	ErrSlugTaken = errors.New("workspace: slug taken")
)

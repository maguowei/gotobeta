package message

import "errors"

var (
	// ErrNotFound 表示消息不存在。
	ErrNotFound = errors.New("message: not found")
	// ErrNotRecallable 表示消息状态不允许撤回。
	ErrNotRecallable = errors.New("message: not recallable")
	// ErrRecallWindowExpired 表示已超过撤回窗口。
	ErrRecallWindowExpired = errors.New("message: recall window expired")
)

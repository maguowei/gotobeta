package message

import "errors"

var (
	// ErrNotFound 表示消息不存在。
	ErrNotFound = errors.New("message: not found")
	// ErrNotRecallable 表示消息状态不允许撤回。
	ErrNotRecallable = errors.New("message: not recallable")
	// ErrRecallWindowExpired 表示已超过撤回窗口。
	ErrRecallWindowExpired = errors.New("message: recall window expired")
	// ErrNotEditable 表示消息类型或状态不允许编辑。
	ErrNotEditable = errors.New("message: not editable")
	// ErrEditWindowExpired 表示已超过编辑窗口。
	ErrEditWindowExpired = errors.New("message: edit window expired")
)

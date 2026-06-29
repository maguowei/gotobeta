package conversation

import "errors"

var (
	// ErrNotFound 表示会话不存在。
	ErrNotFound = errors.New("conversation: not found")
	// ErrMemberNotFound 表示会话成员不存在。
	ErrMemberNotFound = errors.New("conversation: member not found")
	// ErrDMExists 表示该单聊会话已存在。
	ErrDMExists = errors.New("conversation: dm already exists")
)

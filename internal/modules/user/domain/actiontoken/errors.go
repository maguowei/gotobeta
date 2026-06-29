package actiontoken

import "errors"

var (
	// ErrNotFound 表示动作 token 不存在或已失效。
	ErrNotFound = errors.New("action token: not found")
)

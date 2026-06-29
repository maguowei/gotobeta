package oauthstate

import "errors"

var (
	// ErrNotFound 表示 OAuth state 不存在或已失效。
	ErrNotFound = errors.New("oauth state: not found")
)

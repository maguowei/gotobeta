package reaction

import "errors"

// ErrAlreadyExists 表示该用户对该消息的同一 emoji 回应已存在（add 幂等命中）。
var ErrAlreadyExists = errors.New("reaction: already exists")

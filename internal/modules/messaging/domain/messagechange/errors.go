package messagechange

import "errors"

// ErrInvalidChangeType 表示非法的变更类型。
var ErrInvalidChangeType = errors.New("messagechange: invalid change type")

// ErrDuplicateChangeSeq 表示同一会话内 change_seq 撞唯一索引。
// 行锁分配理论零冲突，此错误是并发兜底的可诊断语义。
var ErrDuplicateChangeSeq = errors.New("messagechange: duplicate change_seq")

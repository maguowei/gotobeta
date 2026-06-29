// Package result 定义 workspace 模块用例的应用层结果。
package result

import "time"

// WorkspaceResult 是工作区结果。
type WorkspaceResult struct {
	ID          int64
	Slug        string
	Name        string
	OwnerUserID int64
	Status      int8
	CreatedAt   time.Time
}

// MemberResult 是工作区成员结果。
type MemberResult struct {
	WorkspaceID int64
	UserID      int64
	Status      int8
	JoinedAt    time.Time
}

// RoleResult 是角色结果。
type RoleResult struct {
	ID   int64
	Code string
	Name string
}

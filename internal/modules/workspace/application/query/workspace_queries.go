// Package query 定义 workspace 模块的读用例入参（CQRS 查询）。
package query

// ListMyWorkspacesQuery 列出某用户加入的全部工作区。
type ListMyWorkspacesQuery struct {
	UserID int64
}

// ListRolesQuery 列出工作区内的角色。
type ListRolesQuery struct {
	WorkspaceID    int64
	OperatorUserID int64
}

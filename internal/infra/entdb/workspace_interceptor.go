package entdb

import (
	"context"

	entsql "entgo.io/ent/dialect/sql"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/ent/intercept"
	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
)

// noScopeKey 标记当前 context 显式逃逸工作区隔离。
type noScopeKey struct{}

// WithoutWorkspaceScope 返回一个跳过工作区隔离过滤的 context。
//
// 仅用于明确需要跨工作区的受信路径：平台模板读取、租户初始化复制、
// 后台运维任务。逃逸必须显式、可审计，绝不暴露给普通请求链路。
func WithoutWorkspaceScope(ctx context.Context) context.Context {
	return context.WithValue(ctx, noScopeKey{}, true)
}

func scopeEscaped(ctx context.Context) bool {
	v, ok := ctx.Value(noScopeKey{}).(bool)
	return ok && v
}

// workspaceScopedTypes 是带 workspace_id 列、需要按工作区隔离过滤的实体类型。
// 不含 messages / conversation_members（它们经 conversation 间接隔离，无 workspace_id 列）。
var workspaceScopedTypes = map[string]struct{}{
	ent.TypeWorkspaceMember:         {},
	ent.TypeConversation:            {},
	ent.TypeAttachment:              {},
	ent.TypeBot:                     {},
	ent.TypeRbacRole:                {},
	ent.TypeRbacPermission:          {},
	ent.TypeRbacRolePermission:      {},
	ent.TypeRbacUserRole:            {},
	ent.TypeRbacAclEntry:            {},
	ent.TypeRbacPermissionVersion:   {},
	ent.TypeRbacPermissionChangeLog: {},
}

const workspaceIDColumn = "workspace_id"

// WorkspaceScopeInterceptor 在 context 携带 workspace_id 且未显式逃逸时，
// 对所有带 workspace_id 列的实体查询统一注入 WHERE workspace_id = ? 过滤，
// 作为 DataScope 工作区隔离的 repo 层兜底——即使应用层漏写成员校验，
// 也不会泄漏跨工作区数据。
func WorkspaceScopeInterceptor() ent.Interceptor {
	return intercept.TraverseFunc(func(ctx context.Context, q intercept.Query) error {
		if scopeEscaped(ctx) {
			return nil
		}
		wsID, ok := requestctx.WorkspaceID(ctx)
		if !ok {
			return nil
		}
		if _, scoped := workspaceScopedTypes[q.Type()]; !scoped {
			return nil
		}
		q.WhereP(entsql.FieldEQ(workspaceIDColumn, wsID))
		return nil
	})
}

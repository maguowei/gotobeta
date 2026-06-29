package requestctx

import "context"

type workspaceIDKey struct{}

// WithWorkspaceID 将当前工作区 id 写入上下文。
//
// 该值必须只来自受信来源（鉴权上下文、受信路由段），
// 严禁从请求体读取——它是 DataScope 工作区隔离的依据。
func WithWorkspaceID(ctx context.Context, workspaceID int64) context.Context {
	return context.WithValue(ctx, workspaceIDKey{}, workspaceID)
}

// WorkspaceID 读取当前工作区 id，未设置时返回 (0, false)。
func WorkspaceID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(workspaceIDKey{}).(int64)
	if !ok || id <= 0 {
		return 0, false
	}

	return id, true
}

package authz

import (
	"context"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
)

// AssertWorkspaceScope 是 DataScope 纵深防御第二层：确认 ctx 中受信工作区
// （由 WorkspaceScope 中间件从 path 注入）与命令携带的工作区一致，不一致即越权。
// ctx 未注入工作区时（如创建工作区/内部调用/测试）跳过，由其它层兜底。
func AssertWorkspaceScope(ctx context.Context, cmdWorkspaceID int64) error {
	if ctxWS, ok := requestctx.WorkspaceID(ctx); ok && ctxWS != cmdWorkspaceID {
		return apperr.Forbidden("工作区不一致")
	}
	return nil
}

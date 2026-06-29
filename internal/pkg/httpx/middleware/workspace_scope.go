package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	httpresponse "github.com/maguowei/gotobeta/internal/pkg/httpx/response"
	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
)

// WorkspaceScope 从受信路由段解析工作区 id 并注入请求 context，
// 作为 DataScope 工作区隔离的依据。工作区 id 只来自 path 段，绝不从请求体读取；
// repo 层（entdb 拦截器）据此统一注入 workspace_id 过滤。
func WorkspaceScope(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.Param(paramName)
		wsID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || wsID <= 0 {
			httpresponse.Error(c, apperr.InvalidParam("无效的工作区标识"))
			c.Abort()
			return
		}
		c.Request = c.Request.WithContext(requestctx.WithWorkspaceID(c.Request.Context(), wsID))
		c.Next()
	}
}

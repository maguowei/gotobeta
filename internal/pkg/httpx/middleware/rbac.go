package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	httpresponse "github.com/maguowei/gotobeta/internal/pkg/httpx/response"
	"github.com/maguowei/gotobeta/internal/pkg/rbac"
)

// RequirePermission 要求当前认证主体拥有指定权限。
func RequirePermission(policy rbac.Policy, permission rbac.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := RequireClaims(c)
		if !ok {
			return
		}
		if !policy.Allows(claims.Roles, claims.Permissions, permission) {
			httpresponse.Error(c, apperr.Forbidden("permission denied"))
			c.Abort()
			return
		}
		c.Next()
	}
}

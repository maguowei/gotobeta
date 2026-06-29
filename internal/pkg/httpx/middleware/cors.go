package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 返回最小实现的跨域中间件（不引入第三方依赖）。
// 仅对白名单内的 Origin 回显 CORS 头；"*" 表示放行任意 Origin（此时不带凭证）。
// 预检 OPTIONS 直接以 204 结束。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	set := make(map[string]struct{}, len(allowedOrigins))
	allowAll := false
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
		}
		set[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			_, ok := set[origin]
			if allowAll || ok {
				if allowAll {
					// 通配放行时按规范不能携带凭证，回显 *。
					c.Header("Access-Control-Allow-Origin", "*")
				} else {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Access-Control-Allow-Credentials", "true")
				}
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Methods", strings.Join([]string{
					http.MethodGet, http.MethodPost, http.MethodPut,
					http.MethodPatch, http.MethodDelete, http.MethodOptions,
				}, ", "))
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-Id")
				c.Header("Access-Control-Max-Age", "600")
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

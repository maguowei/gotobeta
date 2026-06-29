package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/httpx/response"
)

// codeRequestEntityTooLarge 是请求体超限响应的业务错误码。
const codeRequestEntityTooLarge = 41301

// BodyLimit 限制请求体大小，抵御超大 body 打爆内存/带宽。
// maxBytes <= 0 表示不限制。已声明 Content-Length 超限时直接 413；
// 同时用 http.MaxBytesReader 兜底分块（chunked）或谎报长度的请求。
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 {
			c.Next()
			return
		}
		if c.Request.ContentLength > maxBytes {
			abortTooLarge(c)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

func abortTooLarge(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, response.APIResponse{
		Code:    codeRequestEntityTooLarge,
		Message: "请求体过大",
		Data:    nil,
	})
}

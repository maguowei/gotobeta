package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
	"github.com/maguowei/gotobeta/internal/pkg/sensitive"
)

const defaultAuditBodyMaxBytes = 64 * 1024

// AuditOptions 是审计日志中间件配置。
type AuditOptions struct {
	Enabled             bool
	LogRequestBody      bool
	LogResponseBody     bool
	MaskSensitiveFields bool
	MaxBodyBytes        int
}

type auditBodyWriter struct {
	gin.ResponseWriter
	body     *bytes.Buffer
	maxBytes int
}

func (w *auditBodyWriter) Write(data []byte) (int, error) {
	if w.body.Len() < w.maxBytes {
		remaining := w.maxBytes - w.body.Len()
		if len(data) <= remaining {
			w.body.Write(data)
		} else {
			w.body.Write(data[:remaining])
		}
	}

	return w.ResponseWriter.Write(data)
}

func (w *auditBodyWriter) WriteString(data string) (int, error) {
	return w.Write([]byte(data))
}

// Audit 记录审计日志。
func Audit(auditLogger *slog.Logger, options AuditOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !options.Enabled {
			c.Next()

			return
		}

		start := time.Now()
		maxBodyBytes := options.MaxBodyBytes
		if maxBodyBytes <= 0 {
			maxBodyBytes = defaultAuditBodyMaxBytes
		}

		requestBody := ""
		if options.LogRequestBody && c.Request.Body != nil {
			raw, readErr := io.ReadAll(io.LimitReader(c.Request.Body, int64(maxBodyBytes)+1))
			c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(raw), c.Request.Body))

			if readErr != nil {
				requestBody = safeAuditReadError()
			} else if len(raw) > maxBodyBytes {
				requestBody = fmt.Sprintf("[body too large, size=%d]", len(raw))
			} else {
				requestBody = maskAuditBody(c.ContentType(), raw, options.MaskSensitiveFields)
			}
		}

		writer := &auditBodyWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
			maxBytes:       maxBodyBytes,
		}
		c.Writer = writer

		c.Next()

		caller, callerType := requestctx.GetCaller(c.Request.Context())
		attrs := []any{
			slog.String("requestId", requestctx.GetRequestID(c.Request.Context())),
			slog.String("caller", caller),
			slog.String("callerType", callerType),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("clientIp", c.ClientIP()),
			slog.String("userAgent", c.Request.UserAgent()),
			slog.Int("statusCode", c.Writer.Status()),
			slog.Int("bizCode", extractBizCode(writer.body.Bytes())),
			slog.Int64("latencyMs", time.Since(start).Milliseconds()),
			slog.Int64("requestSize", c.Request.ContentLength),
			slog.Int("responseSize", writer.body.Len()),
		}

		if options.LogRequestBody {
			attrs = append(attrs, slog.String("requestBody", requestBody))
		}
		if options.LogResponseBody {
			var responseBody string
			if writer.body.Len() >= maxBodyBytes {
				responseBody = fmt.Sprintf("[body too large, size=%d]", writer.body.Len())
			} else {
				responseBody = maskAuditBody(c.Writer.Header().Get("Content-Type"), writer.body.Bytes(), options.MaskSensitiveFields)
			}
			attrs = append(attrs, slog.String("responseBody", responseBody))
		}

		auditLogger.InfoContext(c.Request.Context(), "audit request", attrs...)
	}
}

func safeAuditReadError() string {
	return "[body read error]"
}

func maskAuditBody(contentType string, data []byte, maskSensitiveFields bool) string {
	if len(data) == 0 {
		return ""
	}
	if !maskSensitiveFields {
		return string(data)
	}
	return sensitive.MaskByContentType(contentType, data)
}

func extractBizCode(data []byte) int {
	if len(data) == 0 {
		return -1
	}

	// 只解到顶层键，避免把整个响应体反序列化为 interface 树。
	var body map[string]json.RawMessage
	if err := json.Unmarshal(data, &body); err != nil {
		return -1
	}

	if raw, ok := body["code"]; ok {
		var code int
		if err := json.Unmarshal(raw, &code); err == nil {
			return code
		}
	}

	if raw, ok := body["Status"]; ok {
		var status string
		if err := json.Unmarshal(raw, &status); err == nil {
			if value, err := strconv.Atoi(status); err == nil {
				return value
			}
		}
	}

	return -1
}

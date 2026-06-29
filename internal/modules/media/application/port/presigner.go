// Package port 定义 media 模块对基础设施的出站端口。
package port

import (
	"context"
	"time"
)

// Presigner 提供对象存储预签名直传与公共访问 URL（由 infra/objstore 实现，组合根注入）。
type Presigner interface {
	PresignPut(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
	PublicURL(objectKey string) string
}

// Package objstore 是 S3 兼容对象存储的唯一归口（dev=MinIO / prod=任意 S3 兼容存储）。
//
// 业务模块只依赖 Presigner 端口，不直接 import minio-go SDK（SDK 归口约束）。
package objstore

import (
	"context"
	"time"
)

// Presigner 提供预签名直传与公共访问 URL 能力。
type Presigner interface {
	// PresignPut 为 objectKey 生成有效期 ttl 的预签名 PUT URL，供客户端直传。
	PresignPut(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
	// PublicURL 返回 objectKey 的对外访问 URL（基于配置的 PublicBaseURL）。
	PublicURL(objectKey string) string
}

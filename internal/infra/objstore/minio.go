package objstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/maguowei/gotobeta/internal/infra/config"
)

// MinioPresigner 基于 minio-go 实现 Presigner（适用于任意 S3 兼容存储）。
type MinioPresigner struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

// NewMinioPresigner 创建预签名器。endpoint 为空时返回 (nil, nil)，由调用方决定是否启用附件能力。
func NewMinioPresigner(cfg config.ObjStoreConfig) (*MinioPresigner, error) {
	if cfg.Endpoint == "" {
		return nil, nil
	}
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("objstore: 创建 minio client: %w", err)
	}
	return &MinioPresigner{
		client:        client,
		bucket:        cfg.Bucket,
		publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
	}, nil
}

// PresignPut 生成预签名 PUT URL。
func (p *MinioPresigner) PresignPut(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	u, err := p.client.PresignedPutObject(ctx, p.bucket, objectKey, ttl)
	if err != nil {
		return "", fmt.Errorf("objstore: 生成预签名 URL: %w", err)
	}
	return u.String(), nil
}

// PublicURL 返回对象的对外访问 URL。
func (p *MinioPresigner) PublicURL(objectKey string) string {
	if p.publicBaseURL != "" {
		return p.publicBaseURL + "/" + objectKey
	}
	return p.bucket + "/" + objectKey
}

package refreshtoken

import (
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// RefreshToken 是 refresh token 的支撑性持久化模型（非聚合根）。
//
// 撤销 / 轮换的并发安全由 repository 的原子条件 UPDATE（WHERE revoked_at IS NULL）
// 保证，没有需要在领域内存中强制的不变量；这里只用 New 收敛构造校验，
// 状态字段保持公开供 infra 直接映射。
type RefreshToken struct {
	TokenID           string
	UserID            int64
	TokenHash         string
	ReplacedByTokenID string
	ExpiresAt         time.Time
	RevokedAt         *time.Time
	RevokeReason      string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// New 构造一条新的 refresh token 记录，并收敛构造校验。
func New(tokenID string, userID int64, tokenHash string, expiresAt time.Time, now time.Time) (*RefreshToken, error) {
	if tokenID == "" {
		return nil, apperr.InvalidParam("refresh token ID 不能为空")
	}
	if userID <= 0 {
		return nil, apperr.InvalidParam("refresh token 必须归属有效用户")
	}
	if tokenHash == "" {
		return nil, apperr.InvalidParam("refresh token hash 不能为空")
	}
	if !expiresAt.After(now) {
		return nil, apperr.InvalidParam("refresh token 有效期必须晚于当前时间")
	}
	return &RefreshToken{
		TokenID:   tokenID,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

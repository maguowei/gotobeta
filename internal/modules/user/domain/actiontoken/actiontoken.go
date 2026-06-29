package actiontoken

import (
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

const (
	ActionEmailVerification = "email_verification"
	ActionPasswordReset     = "password_reset"
	ActionOAuthLoginCode    = "oauth_login_code"
)

// ActionToken 是一次性动作 token 的支撑性持久化模型（非聚合根）。
//
// 一次性消费的并发安全由 repository 的原子条件 UPDATE
// （WHERE consumed_at IS NULL AND expires_at > now）保证，没有需要在领域内存中
// 强制的不变量；这里只用 New 收敛构造校验，状态字段保持公开供 infra 直接映射。
type ActionToken struct {
	TokenID               string
	UserID                int64
	Purpose               string
	TokenHash             string
	TargetEmailNormalized string
	ExpiresAt             time.Time
	ConsumedAt            *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// IsValidPurpose 返回 purpose 是否为受支持的动作类型。
func IsValidPurpose(purpose string) bool {
	switch purpose {
	case ActionEmailVerification, ActionPasswordReset, ActionOAuthLoginCode:
		return true
	}
	return false
}

// New 构造一条新的一次性动作 token 记录，并收敛构造校验。
func New(tokenID string, userID int64, purpose string, tokenHash string, targetEmail string, expiresAt time.Time, now time.Time) (*ActionToken, error) {
	if tokenID == "" {
		return nil, apperr.InvalidParam("动作 token ID 不能为空")
	}
	if userID <= 0 {
		return nil, apperr.InvalidParam("动作 token 必须归属有效用户")
	}
	if !IsValidPurpose(purpose) {
		return nil, apperr.InvalidParam("不支持的动作 token 类型")
	}
	if tokenHash == "" {
		return nil, apperr.InvalidParam("动作 token hash 不能为空")
	}
	if !expiresAt.After(now) {
		return nil, apperr.InvalidParam("动作 token 有效期必须晚于当前时间")
	}
	return &ActionToken{
		TokenID:               tokenID,
		UserID:                userID,
		Purpose:               purpose,
		TokenHash:             tokenHash,
		TargetEmailNormalized: targetEmail,
		ExpiresAt:             expiresAt,
		CreatedAt:             now,
		UpdatedAt:             now,
	}, nil
}

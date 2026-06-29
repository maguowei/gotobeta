package user

import (
	"net/mail"
	"strings"
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// User 是认证用户聚合。
type User struct {
	id              int64
	email           string
	emailNormalized string
	emailVerifiedAt *time.Time
	passwordHash    string
	passwordHashAlg string
	passwordSetAt   *time.Time
	displayName     string
	avatarURL       string
	status          Status
	lastLoginAt     *time.Time
	createdAt       time.Time
	updatedAt       time.Time
}

func (u *User) ID() int64                   { return u.id }
func (u *User) Email() string               { return u.email }
func (u *User) EmailNormalized() string     { return u.emailNormalized }
func (u *User) EmailVerifiedAt() *time.Time { return u.emailVerifiedAt }
func (u *User) PasswordHash() string        { return u.passwordHash }
func (u *User) PasswordHashAlg() string     { return u.passwordHashAlg }
func (u *User) PasswordSetAt() *time.Time   { return u.passwordSetAt }
func (u *User) DisplayName() string         { return u.displayName }
func (u *User) AvatarURL() string           { return u.avatarURL }
func (u *User) Status() Status              { return u.status }
func (u *User) LastLoginAt() *time.Time     { return u.lastLoginAt }
func (u *User) CreatedAt() time.Time        { return u.createdAt }
func (u *User) UpdatedAt() time.Time        { return u.updatedAt }

// FromDB 是持久化层重建用户实体的参数。
type FromDB struct {
	ID              int64
	Email           string
	EmailNormalized string
	EmailVerifiedAt *time.Time
	PasswordHash    string
	PasswordHashAlg string
	PasswordSetAt   *time.Time
	DisplayName     string
	AvatarURL       string
	Status          Status
	LastLoginAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// UnmarshalFromDB 从数据库记录重建用户实体，跳过业务校验。仅供 infra 层使用。
func UnmarshalFromDB(raw FromDB) *User {
	return &User{
		id:              raw.ID,
		email:           raw.Email,
		emailNormalized: raw.EmailNormalized,
		emailVerifiedAt: raw.EmailVerifiedAt,
		passwordHash:    raw.PasswordHash,
		passwordHashAlg: raw.PasswordHashAlg,
		passwordSetAt:   raw.PasswordSetAt,
		displayName:     raw.DisplayName,
		avatarURL:       raw.AvatarURL,
		status:          raw.Status,
		lastLoginAt:     raw.LastLoginAt,
		createdAt:       raw.CreatedAt,
		updatedAt:       raw.UpdatedAt,
	}
}

// New 创建邮箱用户。
func New(id int64, email string, displayName string, now time.Time) (*User, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, apperr.InvalidParam("用户 ID 必须大于 0")
	}
	return &User{
		id:              id,
		email:           strings.TrimSpace(email),
		emailNormalized: normalized,
		displayName:     strings.TrimSpace(displayName),
		status:          StatusActive,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// NormalizeEmail 归一化邮箱。
func NormalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return "", apperr.InvalidParam("邮箱不能为空")
	}
	if len(normalized) > 320 {
		return "", apperr.InvalidParam("邮箱长度不能超过 320")
	}
	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", apperr.InvalidParam("邮箱格式不正确")
	}
	return normalized, nil
}

// EnsureCanLogin 校验账号是否允许登录。
func (u *User) EnsureCanLogin() error {
	if u == nil {
		return apperr.Unauthorized("用户不存在")
	}
	if u.status != StatusActive {
		return apperr.Forbidden("账号已停用")
	}
	return nil
}

// SetPassword 写入密码哈希。
func (u *User) SetPassword(hash string, alg string, now time.Time) error {
	if strings.TrimSpace(hash) == "" {
		return apperr.InvalidParam("密码哈希不能为空")
	}
	u.passwordHash = hash
	u.passwordHashAlg = alg
	u.passwordSetAt = &now
	u.updatedAt = now
	return nil
}

// HasPassword 判断用户是否已设置本地密码。
func (u *User) HasPassword() bool {
	return strings.TrimSpace(u.passwordHash) != ""
}

// VerifyEmail 标记邮箱已验证。
func (u *User) VerifyEmail(now time.Time) {
	u.emailVerifiedAt = &now
	u.updatedAt = now
}

// TouchLogin 记录登录时间。
func (u *User) TouchLogin(now time.Time) {
	u.lastLoginAt = &now
	u.updatedAt = now
}

// UpdateProfile 更新基础资料。
func (u *User) UpdateProfile(displayName string, avatarURL string, now time.Time) {
	u.displayName = strings.TrimSpace(displayName)
	u.avatarURL = strings.TrimSpace(avatarURL)
	u.updatedAt = now
}

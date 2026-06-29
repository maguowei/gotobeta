package security

import (
	"golang.org/x/crypto/bcrypt"
)

const bcryptAlgorithm = "bcrypt"

// BcryptPasswordHasher 使用 bcrypt 哈希密码。
type BcryptPasswordHasher struct {
	cost int
}

// NewBcryptPasswordHasher 创建密码哈希器。
func NewBcryptPasswordHasher(cost int) *BcryptPasswordHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptPasswordHasher{cost: cost}
}

// Hash 生成密码哈希。
func (h *BcryptPasswordHasher) Hash(password string) (string, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", "", err
	}
	return string(hash), bcryptAlgorithm, nil
}

// Compare 校验密码。
func (h *BcryptPasswordHasher) Compare(hash string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

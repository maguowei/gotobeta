package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// RandomSecretGenerator 生成随机 token，并只保存哈希。
type RandomSecretGenerator struct{}

// NewRandomSecretGenerator 创建随机 token 生成器。
func NewRandomSecretGenerator() RandomSecretGenerator {
	return RandomSecretGenerator{}
}

// NewToken 生成 URL 安全 token。
func (RandomSecretGenerator) NewToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashToken 返回 SHA-256 hex。
func (RandomSecretGenerator) HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

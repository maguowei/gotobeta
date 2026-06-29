package email

import (
	"context"
	"log/slog"
)

// Sender 发送认证邮件。
type Sender interface {
	SendEmailVerification(ctx context.Context, email string, token string) error
	SendPasswordReset(ctx context.Context, email string, token string) error
}

// LogSender 把邮件 token 写入日志，仅用于本地开发和测试。
type LogSender struct {
	logger *slog.Logger
}

// NewSender 按配置创建邮件发送器。
func NewSender(kind string, logger *slog.Logger) Sender {
	if kind == "disabled" {
		return DisabledSender{}
	}
	return NewLogSender(logger)
}

// NewLogSender 创建开发邮件发送器。
func NewLogSender(logger *slog.Logger) *LogSender {
	return &LogSender{logger: logger}
}

// SendEmailVerification 记录邮箱验证 token。
func (s *LogSender) SendEmailVerification(ctx context.Context, email string, token string) error {
	s.logger.InfoContext(ctx, "email verification token generated", slog.String("email", email), slog.String("token", token))
	return nil
}

// SendPasswordReset 记录密码重置 token。
func (s *LogSender) SendPasswordReset(ctx context.Context, email string, token string) error {
	s.logger.InfoContext(ctx, "password reset token generated", slog.String("email", email), slog.String("token", token))
	return nil
}

// DisabledSender 不发送邮件，也不会把 token 写入日志。
type DisabledSender struct{}

func (DisabledSender) SendEmailVerification(context.Context, string, string) error {
	return nil
}

func (DisabledSender) SendPasswordReset(context.Context, string, string) error {
	return nil
}

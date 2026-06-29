package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/mail"
	"net/smtp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/maguowei/gotobeta/internal/infra/config"
)

// Message 是待发送的纯文本邮件。
type Message struct {
	To       []string
	Subject  string
	TextBody string
}

// Sender 通过 SMTP 发送邮件。配置 disabled 时 Send 是 no-op，方便本地与测试环境复用同一 wiring。
type Sender struct {
	cfg    config.SMTPConfig
	logger *slog.Logger
	dial   smtpDialFunc
}

type smtpDialFunc func(context.Context, config.SMTPConfig) (smtpSession, error)

type smtpSession interface {
	Hello(localName string) error
	StartTLS(config *tls.Config) error
	Auth(auth smtp.Auth) error
	Mail(from string) error
	Rcpt(to string) error
	Data() (io.WriteCloser, error)
	Quit() error
	Close() error
}

// NewSender 创建 SMTP sender。
func NewSender(cfg config.SMTPConfig, logger *slog.Logger) (*Sender, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Sender{cfg: cfg, logger: logger, dial: defaultDialSMTP}, nil
}

// Send 发送纯文本邮件。
func (s *Sender) Send(ctx context.Context, msg Message) error {
	if !s.cfg.Enabled {
		s.logger.DebugContext(ctx, "smtp email sender disabled")
		return nil
	}
	if err := validateMessage(msg); err != nil {
		return err
	}

	session, err := s.dial(ctx, s.cfg)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer func() {
		_ = session.Close()
	}()

	if err := session.Hello("localhost"); err != nil {
		return fmt.Errorf("smtp hello: %w", err)
	}
	if s.cfg.TLSMode == "starttls" {
		if err := session.StartTLS(tlsConfig(s.cfg)); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if strings.TrimSpace(s.cfg.Username) != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := session.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	from, err := parseMailbox(s.cfg.From)
	if err != nil {
		return fmt.Errorf("smtp from: %w", err)
	}
	recipients, err := parseRecipients(msg.To)
	if err != nil {
		return err
	}

	if err := session.Mail(from.Address); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	for _, recipient := range recipients {
		if err := session.Rcpt(recipient.Address); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", recipient.Address, err)
		}
	}

	writer, err := session.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write(formatMessage(from, recipients, msg)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp write message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp close message: %w", err)
	}
	if err := session.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}

func validateConfig(cfg config.SMTPConfig) error {
	if !oneOf(cfg.TLSMode, "none", "starttls", "tls") {
		return errors.New("smtp.tls_mode must be none, starttls, or tls")
	}
	if _, err := time.ParseDuration(cfg.Timeout); err != nil {
		return fmt.Errorf("smtp.timeout: %w", err)
	}
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return errors.New("smtp.host is required")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return errors.New("smtp.port must be between 1 and 65535")
	}
	if _, err := parseMailbox(cfg.From); err != nil {
		return fmt.Errorf("smtp.from: %w", err)
	}
	if strings.TrimSpace(cfg.Username) != "" && strings.TrimSpace(cfg.Password) == "" {
		return errors.New("smtp.password is required when smtp.username is set")
	}
	return nil
}

func validateMessage(msg Message) error {
	if len(msg.To) == 0 {
		return errors.New("email recipients are required")
	}
	if strings.TrimSpace(msg.Subject) == "" {
		return errors.New("email subject is required")
	}
	if containsHeaderBreak(msg.Subject) {
		return errors.New("email subject cannot contain header breaks")
	}
	_, err := parseRecipients(msg.To)
	return err
}

func defaultDialSMTP(ctx context.Context, cfg config.SMTPConfig) (smtpSession, error) {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, err
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	dialer := &net.Dialer{Timeout: timeout}
	var conn net.Conn
	if cfg.TLSMode == "tls" {
		tlsDialer := tls.Dialer{NetDialer: dialer, Config: tlsConfig(cfg)}
		conn, err = tlsDialer.DialContext(ctx, "tcp", addr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func tlsConfig(cfg config.SMTPConfig) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: cfg.Host,
	}
}

func parseRecipients(values []string) ([]mail.Address, error) {
	recipients := make([]mail.Address, 0, len(values))
	for _, value := range values {
		recipient, err := parseMailbox(value)
		if err != nil {
			return nil, fmt.Errorf("email recipient %q: %w", value, err)
		}
		recipients = append(recipients, recipient)
	}
	return recipients, nil
}

func parseMailbox(value string) (mail.Address, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return mail.Address{}, errors.New("email address is required")
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil {
		return mail.Address{}, err
	}
	return *parsed, nil
}

func formatMessage(from mail.Address, recipients []mail.Address, msg Message) []byte {
	toHeaders := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		toHeaders = append(toHeaders, recipient.String())
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", from.String())
	fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(toHeaders, ", "))
	fmt.Fprintf(&buf, "Subject: %s\r\n", msg.Subject)
	fmt.Fprint(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprint(&buf, "Content-Type: text/plain; charset=UTF-8\r\n")
	fmt.Fprint(&buf, "\r\n")
	buf.WriteString(msg.TextBody)
	return buf.Bytes()
}

func containsHeaderBreak(value string) bool {
	return strings.ContainsAny(value, "\r\n")
}

func oneOf(value string, allowed ...string) bool {
	return slices.Contains(allowed, value)
}

package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net/smtp"
	"slices"
	"strings"
	"testing"

	"github.com/maguowei/gotobeta/internal/infra/config"
)

func TestSenderSendUsesStartTLSAndWritesMessage(t *testing.T) {
	session := &fakeSMTPSession{}
	cfg := config.SMTPConfig{
		Enabled:  true,
		Host:     "smtp.example.com",
		Port:     587,
		Username: "mailer",
		Password: "secret",
		From:     "Codego <no-reply@example.com>",
		TLSMode:  "starttls",
		Timeout:  "5s",
	}
	sender := &Sender{
		cfg:    cfg,
		logger: slog.New(slog.DiscardHandler),
		dial: func(context.Context, config.SMTPConfig) (smtpSession, error) {
			return session, nil
		},
	}

	err := sender.Send(t.Context(), Message{
		To:       []string{"Alice <alice@example.com>"},
		Subject:  "Verify email",
		TextBody: "hello",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	for _, want := range []string{"hello:localhost", "starttls", "auth", "mail:no-reply@example.com", "rcpt:alice@example.com", "quit"} {
		if !slices.Contains(session.commands, want) {
			t.Fatalf("commands missing %q: %v", want, session.commands)
		}
	}
	message := session.data.String()
	for _, want := range []string{`From: "Codego" <no-reply@example.com>`, `To: "Alice" <alice@example.com>`, "Subject: Verify email", "hello"} {
		if !strings.Contains(message, want) {
			t.Fatalf("message missing %q:\n%s", want, message)
		}
	}
}

func TestSenderSendNoopsWhenDisabled(t *testing.T) {
	sender, err := NewSender(config.SMTPConfig{Enabled: false, TLSMode: "none", Timeout: "5s"}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewSender() error = %v", err)
	}
	err = sender.Send(t.Context(), Message{To: []string{"alice@example.com"}, Subject: "ignored"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
}

func TestNewSenderRejectsInvalidConfig(t *testing.T) {
	_, err := NewSender(config.SMTPConfig{
		Enabled: true,
		Host:    "smtp.example.com",
		Port:    587,
		From:    "not-an-address",
		TLSMode: "starttls",
		Timeout: "5s",
	}, slog.New(slog.DiscardHandler))
	if err == nil || !strings.Contains(err.Error(), "smtp.from") {
		t.Fatalf("NewSender() error = %v, want smtp.from", err)
	}
}

func TestSenderRejectsHeaderInjection(t *testing.T) {
	sender := &Sender{
		cfg:    config.SMTPConfig{Enabled: true, Host: "smtp.example.com", Port: 587, From: "no-reply@example.com", TLSMode: "none", Timeout: "5s"},
		logger: slog.New(slog.DiscardHandler),
		dial: func(context.Context, config.SMTPConfig) (smtpSession, error) {
			return &fakeSMTPSession{}, nil
		},
	}
	err := sender.Send(t.Context(), Message{To: []string{"alice@example.com"}, Subject: "hi\r\nBcc: victim@example.com"})
	if err == nil || !strings.Contains(err.Error(), "header breaks") {
		t.Fatalf("Send() error = %v, want header break rejection", err)
	}
}

type fakeSMTPSession struct {
	commands []string
	data     bytes.Buffer
}

func (s *fakeSMTPSession) Hello(localName string) error {
	s.commands = append(s.commands, "hello:"+localName)
	return nil
}

func (s *fakeSMTPSession) StartTLS(*tls.Config) error {
	s.commands = append(s.commands, "starttls")
	return nil
}

func (s *fakeSMTPSession) Auth(smtp.Auth) error {
	s.commands = append(s.commands, "auth")
	return nil
}

func (s *fakeSMTPSession) Mail(from string) error {
	s.commands = append(s.commands, "mail:"+from)
	return nil
}

func (s *fakeSMTPSession) Rcpt(to string) error {
	s.commands = append(s.commands, "rcpt:"+to)
	return nil
}

func (s *fakeSMTPSession) Data() (io.WriteCloser, error) {
	s.commands = append(s.commands, "data")
	return nopWriteCloser{Writer: &s.data}, nil
}

func (s *fakeSMTPSession) Quit() error {
	s.commands = append(s.commands, "quit")
	return nil
}

func (s *fakeSMTPSession) Close() error {
	s.commands = append(s.commands, "close")
	return nil
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

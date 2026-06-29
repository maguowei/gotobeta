package email

import (
	"log/slog"
	"testing"
)

func TestLogSender(t *testing.T) {
	sender := NewLogSender(slog.New(slog.DiscardHandler))
	if err := sender.SendEmailVerification(t.Context(), "alice@example.com", "verify-token"); err != nil {
		t.Fatalf("SendEmailVerification() error = %v", err)
	}
	if err := sender.SendPasswordReset(t.Context(), "alice@example.com", "reset-token"); err != nil {
		t.Fatalf("SendPasswordReset() error = %v", err)
	}
}

func TestNewSenderDisabled(t *testing.T) {
	sender := NewSender("disabled", slog.New(slog.DiscardHandler))
	if _, ok := sender.(DisabledSender); !ok {
		t.Fatalf("NewSender(disabled) = %T, want DisabledSender", sender)
	}
	if err := sender.SendEmailVerification(t.Context(), "alice@example.com", "verify-token"); err != nil {
		t.Fatalf("SendEmailVerification() error = %v", err)
	}
	if err := sender.SendPasswordReset(t.Context(), "alice@example.com", "reset-token"); err != nil {
		t.Fatalf("SendPasswordReset() error = %v", err)
	}
}

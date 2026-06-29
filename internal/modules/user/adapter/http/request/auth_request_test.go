package request

import "testing"

func TestRequestToCommand(t *testing.T) {
	if got := (RegisterRequest{Email: "alice@example.com", Password: "password-123", DisplayName: "Alice"}).ToCommand(); got.Email != "alice@example.com" || got.DisplayName != "Alice" {
		t.Fatalf("RegisterRequest.ToCommand() = %+v", got)
	}
	if got := (LoginRequest{Email: "alice@example.com", Password: "password-123"}).ToCommand(); got.Password != "password-123" {
		t.Fatalf("LoginRequest.ToCommand() = %+v", got)
	}
	if got := (RefreshRequest{RefreshToken: "refresh"}).ToCommand(); got.RefreshToken != "refresh" {
		t.Fatalf("RefreshRequest.ToCommand() = %+v", got)
	}
	if got := (LogoutRequest{RefreshToken: "refresh"}).ToCommand(); got.RefreshToken != "refresh" {
		t.Fatalf("LogoutRequest.ToCommand() = %+v", got)
	}
	if got := (ForgotPasswordRequest{Email: "alice@example.com"}).ToCommand(); got.Email != "alice@example.com" {
		t.Fatalf("ForgotPasswordRequest.ToCommand() = %+v", got)
	}
	if got := (ResetPasswordRequest{Token: "token", NewPassword: "password-123"}).ToCommand(); got.Token != "token" {
		t.Fatalf("ResetPasswordRequest.ToCommand() = %+v", got)
	}
	if got := (VerifyEmailRequest{Token: "token"}).ToCommand(); got.Token != "token" {
		t.Fatalf("VerifyEmailRequest.ToCommand() = %+v", got)
	}
	if got := (OAuthTokenRequest{Code: "code"}).ToCommand(); got.Code != "code" {
		t.Fatalf("OAuthTokenRequest.ToCommand() = %+v", got)
	}
	if got := (UpdateProfileRequest{DisplayName: "Alice", AvatarURL: "https://example.com/a.png"}).ToCommand(42); got.UserID != 42 || got.AvatarURL == "" {
		t.Fatalf("UpdateProfileRequest.ToCommand() = %+v", got)
	}
	if got := (ChangePasswordRequest{OldPassword: "old", NewPassword: "new-password"}).ToCommand(42); got.UserID != 42 || got.NewPassword != "new-password" {
		t.Fatalf("ChangePasswordRequest.ToCommand() = %+v", got)
	}
}

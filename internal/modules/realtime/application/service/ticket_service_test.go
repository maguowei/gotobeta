package service

import (
	"context"
	"errors"
	"testing"
)

type fakeTicketStore struct {
	token string
	err   error
}

func (f fakeTicketStore) Issue(_ context.Context, _ int64) (string, error) {
	return f.token, f.err
}
func (f fakeTicketStore) Consume(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func TestIssueTicketOK(t *testing.T) {
	s := NewTicketService(fakeTicketStore{token: "tk-1"})
	token, err := s.IssueTicket(context.Background(), 9)
	if err != nil || token != "tk-1" {
		t.Fatalf("签发失败: %q %v", token, err)
	}
}

func TestIssueTicketErr(t *testing.T) {
	s := NewTicketService(fakeTicketStore{err: errors.New("boom")})
	if _, err := s.IssueTicket(context.Background(), 9); err == nil {
		t.Fatal("底层错误应向上传递")
	}
}

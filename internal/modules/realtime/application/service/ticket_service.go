// Package service 编排 realtime 模块用例（ticket 签发、事件分发）。
package service

import (
	"context"

	"github.com/maguowei/gotobeta/internal/modules/realtime/application/port"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// TicketService 签发 WS 鉴权 ticket。
type TicketService struct {
	tickets port.TicketStore
}

// NewTicketService 创建服务。
func NewTicketService(tickets port.TicketStore) *TicketService {
	return &TicketService{tickets: tickets}
}

// IssueTicket 为登录用户签发一次性 WS ticket。
func (s *TicketService) IssueTicket(ctx context.Context, userID int64) (string, error) {
	token, err := s.tickets.Issue(ctx, userID)
	if err != nil {
		return "", apperr.Internal("签发 ticket 失败", err)
	}
	return token, nil
}

// Package port 定义 realtime 模块对基础设施的出站端口。
package port

import "context"

// TicketStore 签发并一次性消费 WS 鉴权 ticket。
//
// JWT 不入 URL：客户端先用 JWT 换 ticket，再用 ticket 建立 WS 连接。
type TicketStore interface {
	// Issue 为 userID 签发一次性短期 ticket。
	Issue(ctx context.Context, userID int64) (string, error)
	// Consume 校验并消费 ticket，返回其绑定的 userID；无效/过期/已用返回 ErrInvalidTicket。
	Consume(ctx context.Context, token string) (int64, error)
}

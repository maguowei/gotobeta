// Package response 定义 realtime 模块的 HTTP 响应体。
package response

// TicketResponse 是 WS ticket 响应。
type TicketResponse struct {
	Ticket string `json:"ticket"`
}

// ToTicketResponse 构造 ticket 响应。
func ToTicketResponse(token string) TicketResponse {
	return TicketResponse{Ticket: token}
}

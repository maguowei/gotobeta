package response

import "testing"

func TestToTicketResponse(t *testing.T) {
	r := ToTicketResponse("tk-123")
	if r.Ticket != "tk-123" {
		t.Fatalf("ticket 响应错误: %+v", r)
	}
}

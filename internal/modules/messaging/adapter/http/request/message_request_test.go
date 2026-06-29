package request

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// bindSend 用 gin 绑定一段 JSON 到 SendMessageRequest，返回绑定是否成功。
func bindSend(body string) error {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	var req SendMessageRequest
	return c.ShouldBindJSON(&req)
}

func TestSendMessageRequestValidation(t *testing.T) {
	valid := `{"clientMsgId":"c1","contentType":1,"content":{"text":"hi"}}`
	if err := bindSend(valid); err != nil {
		t.Fatalf("合法请求应通过，得 %v", err)
	}

	cases := []struct {
		name string
		body string
	}{
		{"缺 clientMsgId", `{"contentType":1,"content":{"text":"hi"}}`},
		{"clientMsgId 超长", `{"clientMsgId":"` + strings.Repeat("x", 65) + `","contentType":1,"content":{"text":"hi"}}`},
		{"非法 contentType", `{"clientMsgId":"c1","contentType":7,"content":{"text":"hi"}}`},
		{"contentType 为 0", `{"clientMsgId":"c1","contentType":0,"content":{"text":"hi"}}`},
		{"空 content", `{"clientMsgId":"c1","contentType":1,"content":{}}`},
		{"缺 content", `{"clientMsgId":"c1","contentType":1}`},
		{"负 replyToMsgId", `{"clientMsgId":"c1","contentType":1,"content":{"text":"hi"},"replyToMsgId":"-5"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := bindSend(tc.body); err == nil {
				t.Fatalf("非法请求 %q 应被拒绝", tc.name)
			}
		})
	}
}

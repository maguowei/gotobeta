package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiterBlocksAfterBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// burst=3、稳态 60/min（1/s）：连续 3 次放行，第 4 次在同一瞬间应被 429 拦截。
	rl := NewRateLimiter(60, 3)
	frozen := time.Unix(1700000000, 0)
	rl.now = func() time.Time { return frozen }

	engine := gin.New()
	engine.Use(rl.Middleware())
	engine.POST("/auth/login", func(c *gin.Context) { c.Status(http.StatusOK) })

	do := func() int {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", http.NoBody)
		req.RemoteAddr = "203.0.113.7:12345"
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		return rec.Code
	}

	for i := range 3 {
		if code := do(); code != http.StatusOK {
			t.Fatalf("request %d status = %d, want 200", i+1, code)
		}
	}
	if code := do(); code != http.StatusTooManyRequests {
		t.Fatalf("request 4 status = %d, want 429", code)
	}
}

func TestRateLimiterRefillsOverTime(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := NewRateLimiter(60, 1) // 1/s 稳态，burst=1
	frozen := time.Unix(1700000000, 0)
	rl.now = func() time.Time { return frozen }

	if !rl.allow("k") {
		t.Fatal("first allow should pass")
	}
	if rl.allow("k") {
		t.Fatal("second immediate allow should be blocked")
	}

	// 推进 1 秒：补充 1 个令牌，应再次放行。
	frozen = frozen.Add(time.Second)
	if !rl.allow("k") {
		t.Fatal("allow after 1s refill should pass")
	}
}

func TestRateLimiterIsolatesKeys(t *testing.T) {
	rl := NewRateLimiter(60, 1)
	frozen := time.Unix(1700000000, 0)
	rl.now = func() time.Time { return frozen }

	if !rl.allow("ip-a") {
		t.Fatal("ip-a first allow should pass")
	}
	// 不同 key 各自独立计数，ip-b 不应受 ip-a 影响。
	if !rl.allow("ip-b") {
		t.Fatal("ip-b first allow should pass")
	}
}

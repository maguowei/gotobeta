package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newLimiterRouter(l *Limiter, keyFunc func(*gin.Context) string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", l.Middleware(keyFunc), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestLimiterAllowsWithinBurst(t *testing.T) {
	r := newLimiterRouter(NewLimiter(60, 3), nil)
	for i := range 3 {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1111"
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("第 %d 次请求应放行，得 %d", i+1, w.Code)
		}
	}
}

func TestLimiterRejectsOverBurst(t *testing.T) {
	r := newLimiterRouter(NewLimiter(60, 2), nil)
	codes := make([]int, 0, 3)
	for range 3 {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "5.6.7.8:2222"
		r.ServeHTTP(w, req)
		codes = append(codes, w.Code)
	}
	if codes[2] != http.StatusTooManyRequests {
		t.Fatalf("超过 burst 应返回 429，得 %v", codes)
	}
}

func TestLimiterRefillsOverTime(t *testing.T) {
	l := NewLimiter(60, 1) // 1/s 稳态，burst=1
	frozen := time.Unix(1700000000, 0)
	l.now = func() time.Time { return frozen }

	if !l.allow("k") {
		t.Fatal("首次应放行")
	}
	if l.allow("k") {
		t.Fatal("同一瞬间第二次应被拒")
	}
	// 推进 1 秒补充 1 个令牌，应再次放行。
	frozen = frozen.Add(time.Second)
	if !l.allow("k") {
		t.Fatal("补充后应放行")
	}
}

func TestLimiterPartitionsByKeyFunc(t *testing.T) {
	keyFunc := func(c *gin.Context) string { return c.GetHeader("X-User") }
	r := newLimiterRouter(NewLimiter(60, 1), keyFunc)

	// 用户 A 用满配额。
	wa1 := httptest.NewRecorder()
	ra1 := httptest.NewRequest(http.MethodGet, "/", nil)
	ra1.Header.Set("X-User", "a")
	r.ServeHTTP(wa1, ra1)

	wa2 := httptest.NewRecorder()
	ra2 := httptest.NewRequest(http.MethodGet, "/", nil)
	ra2.Header.Set("X-User", "a")
	r.ServeHTTP(wa2, ra2)
	if wa2.Code != http.StatusTooManyRequests {
		t.Fatalf("用户 A 超额应 429，得 %d", wa2.Code)
	}

	// 用户 B 独立分桶，不受 A 影响。
	wb := httptest.NewRecorder()
	rb := httptest.NewRequest(http.MethodGet, "/", nil)
	rb.Header.Set("X-User", "b")
	r.ServeHTTP(wb, rb)
	if wb.Code != http.StatusOK {
		t.Fatalf("用户 B 应独立放行，得 %d", wb.Code)
	}
}

package router

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/handler"
	"github.com/maguowei/gotobeta/internal/pkg/auth"
	"github.com/maguowei/gotobeta/internal/pkg/httpx/middleware"
)

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	// 仅校验注册不 panic；handler 为 nil 时方法值延迟到调用才解引用。
	RegisterRoutes(e.Group("/api/v1"), &handler.ConversationHandler{}, &handler.MessageHandler{}, nil, func(c *gin.Context) { c.Next() })
	if len(e.Routes()) == 0 {
		t.Fatal("应注册路由")
	}
}

// withUID 注入带指定 UserID 的认证 claims，模拟登录态。
func withUID(uid int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if uid > 0 {
			c.Request = c.Request.WithContext(auth.WithClaims(c.Request.Context(), &auth.Claims{UserID: uid}))
		}
		c.Next()
	}
}

func TestSendRateLimitPartitionsByUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 同一 limiter 实例（burst=1），区分用户：用户 1 超额 429，用户 2 仍放行。
	l := middleware.NewLimiter(60, 1)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		uid, _ := strconv.ParseInt(c.GetHeader("X-UID"), 10, 64)
		withUID(uid)(c)
	}, l.Middleware(UserRateKey), func(c *gin.Context) { c.Status(http.StatusOK) })

	do := func(uid string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-UID", uid)
		r.ServeHTTP(w, req)
		return w.Code
	}

	if c := do("1"); c != http.StatusOK {
		t.Fatalf("用户1首次应放行，得 %d", c)
	}
	if c := do("1"); c != http.StatusTooManyRequests {
		t.Fatalf("用户1超额应 429，得 %d", c)
	}
	if c := do("2"); c != http.StatusOK {
		t.Fatalf("用户2独立分桶应放行，得 %d", c)
	}
}

func TestUserRateKeyFallsBackWithoutClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	if got := UserRateKey(c); got != "" {
		t.Fatalf("无 claims 时应返回空串（中间件按 IP 兜底），得 %q", got)
	}
}

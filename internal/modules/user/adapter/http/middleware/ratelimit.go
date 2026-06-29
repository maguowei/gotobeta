package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/httpx/response"
)

// codeTooManyRequests 是限流响应使用的业务错误码。
const codeTooManyRequests = 42901

// RateLimiter 是按客户端 IP 维度的令牌桶限流器，用于抵御认证端点的密码爆破/撞库。
// 采用惰性清理（无后台 goroutine），避免 goroutine 泄漏。
type RateLimiter struct {
	mu          sync.Mutex
	visitors    map[string]*visitor
	ratePerSec  float64
	burst       float64
	ttl         time.Duration
	lastCleanup time.Time
	now         func() time.Time
}

type visitor struct {
	tokens   float64
	lastSeen time.Time
}

// NewRateLimiter 构造限流器。requestsPerMinute 为稳态速率，burst 为瞬时突发容量。
func NewRateLimiter(requestsPerMinute, burst int) *RateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 60
	}
	if burst <= 0 {
		burst = 10
	}

	return &RateLimiter{
		visitors:   make(map[string]*visitor),
		ratePerSec: float64(requestsPerMinute) / 60.0,
		burst:      float64(burst),
		ttl:        10 * time.Minute,
		now:        time.Now,
	}
}

// allow 判断给定 key 是否还有令牌可用，并扣减一个令牌。
func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	rl.cleanup(now)

	v, ok := rl.visitors[key]
	if !ok {
		// 首次访问：发放满桶并扣减本次请求。
		rl.visitors[key] = &visitor{tokens: rl.burst - 1, lastSeen: now}
		return true
	}

	// 按经过时间补充令牌，上限为 burst。
	v.tokens += now.Sub(v.lastSeen).Seconds() * rl.ratePerSec
	if v.tokens > rl.burst {
		v.tokens = rl.burst
	}
	v.lastSeen = now

	if v.tokens < 1 {
		return false
	}
	v.tokens--

	return true
}

// cleanup 惰性清除长期不活跃的 visitor，限制内存占用。调用方需持有锁。
func (rl *RateLimiter) cleanup(now time.Time) {
	if now.Sub(rl.lastCleanup) < rl.ttl {
		return
	}
	for key, v := range rl.visitors {
		if now.Sub(v.lastSeen) > rl.ttl {
			delete(rl.visitors, key)
		}
	}
	rl.lastCleanup = now
}

// Middleware 返回 gin 中间件，超过限流阈值时以 429 拒绝请求。
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, response.APIResponse{
				Code:    codeTooManyRequests,
				Message: "请求过于频繁，请稍后再试",
				Data:    nil,
			})

			return
		}
		c.Next()
	}
}

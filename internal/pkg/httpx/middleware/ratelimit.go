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

// Limiter 是按 key 维度的令牌桶限流器，可复用于按 IP（认证端点防爆破）或按用户
// （发消息频控）等场景。采用惰性清理（无后台 goroutine），避免 goroutine 泄漏。
type Limiter struct {
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

// NewLimiter 构造限流器。requestsPerMinute 为稳态速率，burst 为瞬时突发容量。
func NewLimiter(requestsPerMinute, burst int) *Limiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 60
	}
	if burst <= 0 {
		burst = 10
	}

	return &Limiter{
		visitors:   make(map[string]*visitor),
		ratePerSec: float64(requestsPerMinute) / 60.0,
		burst:      float64(burst),
		ttl:        10 * time.Minute,
		now:        time.Now,
	}
}

// allow 判断给定 key 是否还有令牌可用，并扣减一个令牌。
func (l *Limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.cleanup(now)

	v, ok := l.visitors[key]
	if !ok {
		// 首次访问：发放满桶并扣减本次请求。
		l.visitors[key] = &visitor{tokens: l.burst - 1, lastSeen: now}
		return true
	}

	// 按经过时间补充令牌，上限为 burst。
	v.tokens += now.Sub(v.lastSeen).Seconds() * l.ratePerSec
	if v.tokens > l.burst {
		v.tokens = l.burst
	}
	v.lastSeen = now

	if v.tokens < 1 {
		return false
	}
	v.tokens--

	return true
}

// cleanup 惰性清除长期不活跃的 visitor，限制内存占用。调用方需持有锁。
func (l *Limiter) cleanup(now time.Time) {
	if now.Sub(l.lastCleanup) < l.ttl {
		return
	}
	for key, v := range l.visitors {
		if now.Sub(v.lastSeen) > l.ttl {
			delete(l.visitors, key)
		}
	}
	l.lastCleanup = now
}

// Middleware 返回 gin 中间件，超过限流阈值时以 429 拒绝请求。
// keyFunc 为 nil 时按客户端 IP 分桶；否则按其返回值分桶（如用户 ID）。
func (l *Limiter) Middleware(keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if keyFunc != nil {
			if k := keyFunc(c); k != "" {
				key = k
			}
		}
		if !l.allow(key) {
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

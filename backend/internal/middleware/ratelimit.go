package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// IPRateLimit is a fixed-window per-IP rate limiter. Used to protect public
// endpoints that proxy to upstream services (e.g. weather → Open-Meteo) from
// being abused as an amplifier — without this, a malicious client could spam
// requests with random query params, blow past the cache, and get the
// server's IP banned by upstream.
//
// Memory: keeps one bucket per IP. Buckets are evicted on access if their
// window has expired by 5×; for a panel with bounded user count this is
// negligible. No background goroutine.
type IPRateLimit struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	bucks  map[string]*ipBucket
}

type ipBucket struct {
	count   int
	resetAt time.Time
}

// NewIPRateLimit returns a limiter that allows max requests per window per IP.
func NewIPRateLimit(max int, window time.Duration) *IPRateLimit {
	return &IPRateLimit{
		max:    max,
		window: window,
		bucks:  make(map[string]*ipBucket),
	}
}

// Middleware returns a Gin handler. 429 with Retry-After header on rejection.
func (l *IPRateLimit) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		retryAfter, ok := l.allow(ip)
		if !ok {
			c.Header("Retry-After", retryAfter.String())
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": 429,
				"msg":  "rate limit exceeded; retry after " + retryAfter.String(),
			})
			return
		}
		c.Next()
	}
}

func (l *IPRateLimit) allow(ip string) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()

	// Opportunistic cleanup: if a bucket is way past its reset, evict it.
	// Saves us a background goroutine. Keeps the map small under churn.
	for k, b := range l.bucks {
		if now.Sub(b.resetAt) > 5*l.window {
			delete(l.bucks, k)
		}
	}

	b, ok := l.bucks[ip]
	if !ok || now.After(b.resetAt) {
		l.bucks[ip] = &ipBucket{count: 1, resetAt: now.Add(l.window)}
		return 0, true
	}
	if b.count >= l.max {
		return time.Until(b.resetAt), false
	}
	b.count++
	return 0, true
}

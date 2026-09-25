package middleware

import (
	"net/http"
	"sync"
	"time"

	"hirescope/backend/internal/handler"

	"github.com/gin-gonic/gin"
)

type clientRecord struct {
	count       int
	windowStart time.Time
}

// IPRateLimiter provides lightweight, thread-safe in-memory rate limiting per client IP.
type IPRateLimiter struct {
	mu           sync.Mutex
	clients      map[string]*clientRecord
	maxRequests  int
	windowLength time.Duration
}

// NewIPRateLimiter creates a new rate limiter with specified max requests per window duration.
func NewIPRateLimiter(maxRequests int, windowLength time.Duration) *IPRateLimiter {
	limiter := &IPRateLimiter{
		clients:      make(map[string]*clientRecord),
		maxRequests:  maxRequests,
		windowLength: windowLength,
	}

	// Periodically cleanup expired entries to prevent memory accumulation
	go func() {
		ticker := time.NewTicker(windowLength * 2)
		for range ticker.C {
			limiter.mu.Lock()
			now := time.Now()
			for ip, rec := range limiter.clients {
				if now.Sub(rec.windowStart) > limiter.windowLength {
					delete(limiter.clients, ip)
				}
			}
			limiter.mu.Unlock()
		}
	}()

	return limiter
}

// Limit creates a Gin middleware that enforces the rate limit.
func (l *IPRateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		l.mu.Lock()
		rec, exists := l.clients[ip]
		if !exists || now.Sub(rec.windowStart) > l.windowLength {
			l.clients[ip] = &clientRecord{
				count:       1,
				windowStart: now,
			}
			l.mu.Unlock()
			c.Next()
			return
		}

		if rec.count >= l.maxRequests {
			l.mu.Unlock()
			handler.RespondError(c, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "too many requests, please try again later")
			c.Abort()
			return
		}

		rec.count++
		l.mu.Unlock()
		c.Next()
	}
}

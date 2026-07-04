package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter stores per-IP rate limiters with periodic cleanup.
type IPRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*ipEntry
	rate     rate.Limit
	burst    int
	stopCh   chan struct{}
}

// NewIPRateLimiter creates a per-IP rate limiter.
// rate defines max requests per second per IP; burst allows short spikes.
// Cleanup runs every cleanupInterval to remove stale entries.
func NewIPRateLimiter(r rate.Limit, burst int, cleanupInterval time.Duration) *IPRateLimiter {
	rl := &IPRateLimiter{
		visitors: make(map[string]*ipEntry),
		rate:     r,
		burst:    burst,
		stopCh:   make(chan struct{}),
	}
	go rl.cleanup(cleanupInterval)
	return rl
}

// Stop halts the cleanup goroutine.
func (rl *IPRateLimiter) Stop() {
	close(rl.stopCh)
}

// getLimiter returns (or creates) the rate limiter for the given IP.
func (rl *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.visitors[ip]
	if !exists {
		entry = &ipEntry{
			limiter:  rate.NewLimiter(rl.rate, rl.burst),
			lastSeen: time.Now(),
		}
		rl.visitors[ip] = entry
	} else {
		entry.lastSeen = time.Now()
	}
	return entry.limiter
}

// cleanup periodically removes stale entries that haven't been used recently.
func (rl *IPRateLimiter) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			for ip, entry := range rl.visitors {
				if time.Since(entry.lastSeen) > interval*2 {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}

// RateLimitMiddleware returns an Echo middleware that rate-limits by client IP.
func RateLimitMiddleware(rl *IPRateLimiter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			limiter := rl.getLimiter(ip)
			if !limiter.Allow() {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "Too many requests. Please try again later.",
				})
			}
			return next(c)
		}
	}
}

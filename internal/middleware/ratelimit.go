package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RateLimiter implements token bucket algorithm for rate limiting
type RateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     int           // requests per window
	window   time.Duration // time window
}

type Visitor struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// rate: max requests per window (e.g., 100)
// window: time window (e.g., 1 minute)
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		window:   window,
	}

	// Cleanup old visitors every 5 minutes
	go rl.cleanup()

	return rl
}

// Middleware returns a Fiber middleware handler
func (rl *RateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get client identifier (IP address)
		ip := c.IP()

		// Allow health checks to bypass rate limiting
		if c.Path() == "/health" {
			return c.Next()
		}

		if !rl.allow(ip) {
			c.Set("X-RateLimit-Limit", string(rune(rl.rate)))
			c.Set("X-RateLimit-Remaining", "0")
			c.Set("Retry-After", string(rune(int(rl.window.Seconds()))))

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "rate limit exceeded",
				"message":     "Too many requests. Please try again later.",
				"retry_after": int(rl.window.Seconds()),
			})
		}

		return c.Next()
	}
}

// allow checks if request is allowed based on rate limit
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	visitor, exists := rl.visitors[ip]
	if !exists {
		visitor = &Visitor{
			tokens:     rl.rate,
			lastRefill: time.Now(),
		}
		rl.visitors[ip] = visitor
	}
	rl.mu.Unlock()

	visitor.mu.Lock()
	defer visitor.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(visitor.lastRefill)

	// Refill tokens based on elapsed time
	if elapsed >= rl.window {
		visitor.tokens = rl.rate
		visitor.lastRefill = now
	} else {
		// Partial refill based on time passed
		tokensToAdd := int(float64(rl.rate) * (elapsed.Seconds() / rl.window.Seconds()))
		visitor.tokens += tokensToAdd
		if visitor.tokens > rl.rate {
			visitor.tokens = rl.rate
		}
		if tokensToAdd > 0 {
			visitor.lastRefill = now
		}
	}

	// Check if request is allowed
	if visitor.tokens > 0 {
		visitor.tokens--
		return true
	}

	return false
}

// cleanup removes inactive visitors
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, visitor := range rl.visitors {
			visitor.mu.Lock()
			if now.Sub(visitor.lastRefill) > rl.window*2 {
				delete(rl.visitors, ip)
			}
			visitor.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

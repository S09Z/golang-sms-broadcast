package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// SecurityHeaders applies OWASP recommended security headers
func SecurityHeaders() fiber.Handler {
	return helmet.New(helmet.Config{
		// X-XSS-Protection: Prevents XSS attacks
		XSSProtection: "1; mode=block",

		// X-Content-Type-Options: Prevents MIME sniffing
		ContentTypeNosniff: "nosniff",

		// X-Frame-Options: Prevents clickjacking
		XFrameOptions: "SAMEORIGIN",

		// Strict-Transport-Security: Enforces HTTPS
		// OWASP: Always use HTTPS in production
		HSTSMaxAge: 31536000, // 1 year

		// Content-Security-Policy: Restricts resource loading
		// OWASP: Defense against XSS and data injection
		ContentSecurityPolicy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';",

		// Referrer-Policy: Controls referrer information
		// OWASP: Protects against information leakage
		ReferrerPolicy: "strict-origin-when-cross-origin",
	})
}

// RequestIDMiddleware adds unique request ID for tracing
func RequestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if request ID already exists
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("X-Request-ID", requestID)
		c.Locals("request_id", requestID)
		return c.Next()
	}
}

// DDoSProtection applies additional DDoS protection
func DDoSProtection() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        100,             // Max requests per window
		Expiration: 1 * time.Minute, // Time window
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP() // Rate limit by IP
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "too many requests",
				"message": "DDoS protection triggered. Please try again later.",
			})
		},
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
	})
}

// generateRequestID generates a simple request ID
func generateRequestID() string {
	// Use a simple timestamp-based ID
	// In production, consider using UUID or more sophisticated method
	return time.Now().Format("20060102150405.000000")
}

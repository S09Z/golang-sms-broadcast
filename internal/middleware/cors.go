package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORSConfig returns CORS configuration following OWASP 2026 principles
func CORSConfig() fiber.Handler {
	return cors.New(cors.Config{
		// AllowOrigins: Explicitly define allowed origins
		// OWASP: Never use "*" in production with credentials
		AllowOrigins: getAllowedOrigins(),

		// AllowMethods: Only allow necessary HTTP methods
		// OWASP: Minimize attack surface by restricting methods
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",

		// AllowHeaders: Specify allowed request headers
		// OWASP: Whitelist only required headers
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Request-ID",

		// AllowCredentials: Enable if using cookies/auth
		// OWASP: Only enable if necessary and with specific origins
		AllowCredentials: false,

		// ExposeHeaders: Headers that client can access
		// OWASP: Only expose necessary headers
		ExposeHeaders: "Content-Length,X-Request-ID",

		// MaxAge: Cache preflight requests (in seconds)
		// OWASP: Reasonable cache time to reduce preflight requests
		MaxAge: 3600, // 1 hour
	})
}

// getAllowedOrigins returns list of allowed origins
// In production, load from environment variables
func getAllowedOrigins() string {
	// Default: localhost for development
	// Production: Replace with actual domains
	return "http://localhost:3000,http://localhost:8080,http://127.0.0.1:3000"

	// Production example:
	// return os.Getenv("ALLOWED_ORIGINS") // "https://app.example.com,https://admin.example.com"
}

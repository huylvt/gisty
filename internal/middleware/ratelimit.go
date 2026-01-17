package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

const (
	// DefaultRateLimit is the default rate limit per minute
	DefaultRateLimit = 5
	// DefaultRatePeriod is the default rate limiting period
	DefaultRatePeriod = time.Minute
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// RequestsPerMinute is the maximum number of requests per minute per IP
	RequestsPerMinute int
	// Enabled controls whether rate limiting is active
	Enabled bool
}

// RateLimiter wraps the limiter instance
type RateLimiter struct {
	limiter *limiter.Limiter
	config  RateLimitConfig
}

// NewRateLimiter creates a new RateLimiter with the given configuration
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	cfg := RateLimitConfig{
		RequestsPerMinute: DefaultRateLimit,
		Enabled:           true,
	}

	if config != nil {
		if config.RequestsPerMinute > 0 {
			cfg.RequestsPerMinute = config.RequestsPerMinute
		}
		cfg.Enabled = config.Enabled
	}

	// Create rate using format "requests-period"
	rate := limiter.Rate{
		Period: DefaultRatePeriod,
		Limit:  int64(cfg.RequestsPerMinute),
	}

	// Use in-memory store
	store := memory.NewStore()

	return &RateLimiter{
		limiter: limiter.New(store, rate),
		config:  cfg,
	}
}

// Middleware returns a Gin middleware that applies rate limiting
func (r *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if rate limiting is disabled
		if !r.config.Enabled {
			c.Next()
			return
		}

		// Get client IP
		ip := c.ClientIP()

		// Get limiter context
		ctx, err := r.limiter.Get(c.Request.Context(), ip)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Rate limiter error",
			})
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.FormatInt(ctx.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(ctx.Remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(ctx.Reset, 10))

		// Check if rate limit exceeded
		if ctx.Reached {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": ctx.Reset - time.Now().Unix(),
			})
			return
		}

		c.Next()
	}
}

// GetConfig returns the current rate limit configuration
func (r *RateLimiter) GetConfig() RateLimitConfig {
	return r.config
}
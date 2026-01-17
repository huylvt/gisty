package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
)

const (
	// MaxContentSize is the maximum allowed content size (1MB)
	MaxContentSize = 1 * 1024 * 1024 // 1MB

	// MaxRequestBodySize is the maximum allowed request body size (1MB + overhead for JSON)
	MaxRequestBodySize = MaxContentSize + 1024 // 1MB + 1KB for JSON overhead
)

// Sanitizer provides input sanitization functionality
type Sanitizer struct {
	policy *bluemonday.Policy
}

// NewSanitizer creates a new Sanitizer instance
func NewSanitizer() *Sanitizer {
	// Use StrictPolicy - strips all HTML
	// This is safe for a code/text paste service
	policy := bluemonday.StrictPolicy()

	return &Sanitizer{
		policy: policy,
	}
}

// SanitizeHTML removes all HTML tags from the input
func (s *Sanitizer) SanitizeHTML(input string) string {
	return s.policy.Sanitize(input)
}

// ContentSizeMiddleware limits the request body size
func ContentSizeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check Content-Length header first for quick rejection
		if c.Request.ContentLength > MaxRequestBodySize {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":    "Content too large",
				"max_size": "1MB",
			})
			return
		}

		// Limit the request body reader
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxRequestBodySize)

		c.Next()
	}
}

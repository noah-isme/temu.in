package middleware

import (
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"temu.in/backend/internal/logging"
)

// RequestIDKey is the gin context key holding the request ID.
const RequestIDKey = "request_id"

// RequestIDHeader is the header read from requests and echoed on responses.
const RequestIDHeader = "X-Request-ID"

var safeRequestID = regexp.MustCompile(`^[A-Za-z0-9._\-]{1,128}$`)

// RequestID accepts a sane incoming X-Request-ID or generates a UUID, then
// exposes it on the gin context, the request context (for slog) and the response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if !safeRequestID.MatchString(id) {
			id = uuid.NewString()
		}
		c.Set(RequestIDKey, id)
		c.Header(RequestIDHeader, id)
		c.Request = c.Request.WithContext(logging.WithRequestID(c.Request.Context(), id))
		c.Next()
	}
}

// RequestIDFromGin returns the request ID set by RequestID, or "".
func RequestIDFromGin(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return c.GetString(RequestIDKey)
}

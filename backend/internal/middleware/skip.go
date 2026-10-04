package middleware

import "github.com/gin-gonic/gin"

// HealthPathPrefixes are the probe paths exempt from rate limiting.
var HealthPathPrefixes = []string{"/health/", "/api/health/"}

// SkipPaths runs next only for requests whose URL path does not start with any
// of prefixes; matching requests continue straight down the chain.
func SkipPaths(prefixes []string, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if hasAnyPrefix(c.Request.URL.Path, prefixes) {
			c.Next()
			return
		}
		next(c)
	}
}

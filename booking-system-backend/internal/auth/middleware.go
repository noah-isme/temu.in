package auth

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

const UserContextKey = "user_claims"

func Middleware(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        auth := c.GetHeader("Authorization")
        if auth == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing auth"})
            return
        }
        parts := strings.SplitN(auth, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid auth header"})
            return
        }

        claims, err := ParseToken(secret, parts[1])
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            return
        }

        c.Set(UserContextKey, claims)
        c.Next()
    }
}

// RequireRole returns middleware that checks the JWT role claim
func RequireRole(role string) gin.HandlerFunc {
    return func(c *gin.Context) {
        v, ok := c.Get(UserContextKey)
        if !ok {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "missing claims"})
            return
        }
        claims, ok := v.(*Claims)
        if !ok {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid claims"})
            return
        }
        if claims.Role != role {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
            return
        }
        c.Next()
    }
}

package Middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"web_backend/Repository/AuthRepositorys"
)

// RequireAuth validates the "Authorization: Bearer <token>" header against
// the access-token JWT and rejects the request with 401 if missing/invalid.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")
		userID, err := AuthRepositorys.ValidateAccessToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

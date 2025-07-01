package middleware

import (
	"net/http"
	"strings"

	"clipflow/auth"
	"clipflow/models"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT tokens and sets user context
func AuthMiddleware(db *models.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Check if the header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate the token
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Get user from database
		user, err := db.GetUserByID(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		// Set user in context
		c.Set("user", user)
		c.Set("userID", user.ID)
		c.Next()
	}
}

// OptionalAuthMiddleware validates JWT tokens if present but doesn't require them
func OptionalAuthMiddleware(db *models.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		user, err := db.GetUserByID(claims.UserID)
		if err != nil {
			c.Next()
			return
		}

		c.Set("user", user)
		c.Set("userID", user.ID)
		c.Next()
	}
}

package middleware

import (
	"log"
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
		log.Printf("🔐 [Middleware] Authorization header: %s", authHeader)

		if authHeader == "" {
			log.Printf("🔐 [Middleware] No Authorization header found, proceeding without auth")
			c.Next()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Printf("🔐 [Middleware] Invalid Authorization header format (missing 'Bearer ' prefix)")
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		log.Printf("🔐 [Middleware] Token string: %s...", tokenString[:min(len(tokenString), 20)])

		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			log.Printf("🔐 [Middleware] Token validation failed: %v", err)
			c.Next()
			return
		}
		log.Printf("🔐 [Middleware] Token validated successfully, userID: %s", claims.UserID)

		user, err := db.GetUserByID(claims.UserID)
		if err != nil {
			log.Printf("🔐 [Middleware] Failed to get user from database: %v", err)
			log.Printf("🔐 [Middleware] UserID from token not found in database, proceeding without auth")
			c.Next()
			return
		}
		log.Printf("🔐 [Middleware] User found in database: %s (%s)", user.ID, user.Email)

		c.Set("user", user)
		c.Set("userID", user.ID)
		log.Printf("🔐 [Middleware] User context set successfully for path: %s", c.Request.URL.Path)
		c.Next()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

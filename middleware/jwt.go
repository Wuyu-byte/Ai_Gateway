package middleware

import (
	"net/http"
	"strings"

	"ai-gateway/model"
	"ai-gateway/service"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserKey   = "current_user"
	ContextAPIKeyKey = "current_api_key"
)

func JWTMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing or invalid Authorization header",
			})
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		claims, err := authService.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid jwt token",
			})
			return
		}

		user, err := authService.CurrentUser(c.Request.Context(), claims.UserID)
		if err != nil || user.Status != "active" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "user not found or disabled",
			})
			return
		}

		c.Set(ContextUserKey, user)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (*model.User, bool) {
	value, exists := c.Get(ContextUserKey)
	if !exists {
		return nil, false
	}

	user, ok := value.(*model.User)
	return user, ok
}

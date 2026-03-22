package middleware

import (
	"net/http"
	"strings"

	"ai-gateway/model"
	"ai-gateway/service"

	"github.com/gin-gonic/gin"
)

func APIKeyMiddleware(apiKeyService *service.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing or invalid Authorization header",
			})
			return
		}

		rawKey := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		apiKey, user, err := apiKeyService.Authenticate(c.Request.Context(), rawKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.Set(ContextUserKey, user)
		c.Set(ContextAPIKeyKey, apiKey)
		c.Next()
	}
}

func CurrentAPIKey(c *gin.Context) (*model.APIKey, bool) {
	value, exists := c.Get(ContextAPIKeyKey)
	if !exists {
		return nil, false
	}

	apiKey, ok := value.(*model.APIKey)
	return apiKey, ok
}

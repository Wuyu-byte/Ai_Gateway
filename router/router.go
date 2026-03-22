package router

import (
	"net/http"

	"ai-gateway/api"
	"ai-gateway/config"
	"ai-gateway/metrics"
	"ai-gateway/middleware"
	"ai-gateway/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func NewRouter(
	cfg *config.Config,
	authService *service.AuthService,
	apiKeyService *service.APIKeyService,
	redisClient *redis.Client,
	collector *metrics.Collector,
	authHandler *api.AuthHandler,
	apiKeyHandler *api.APIKeyHandler,
	chatHandler *api.ChatHandler,
	statsHandler *api.StatsHandler,
) *gin.Engine {
	gin.SetMode(gin.DebugMode)
	if cfg.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())
	if collector != nil {
		engine.Use(collector.Middleware())
		engine.GET("/metrics", gin.WrapH(collector.Handler()))
	}

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	auth := engine.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)

	apikey := engine.Group("/apikey")
	apikey.Use(middleware.JWTMiddleware(authService))
	apikey.POST("/create", apiKeyHandler.Create)
	apikey.GET("/list", apiKeyHandler.List)
	apikey.DELETE("/:id", apiKeyHandler.Delete)

	manage := engine.Group("/v1")
	manage.Use(middleware.JWTMiddleware(authService))
	manage.GET("/stats/usage/daily", statsHandler.DailyUsage)
	manage.GET("/stats/usage/users", statsHandler.UserUsage)
	manage.GET("/stats/usage/providers", statsHandler.ProviderUsage)
	manage.GET("/providers/status", statsHandler.ProviderStatus)

	v1 := engine.Group("/v1")
	v1.Use(middleware.APIKeyMiddleware(apiKeyService))
	v1.Use(middleware.RateLimitMiddleware(redisClient, cfg.RateLimit.PerMinute))
	v1.POST("/chat/completions", chatHandler.ChatCompletions)

	return engine
}

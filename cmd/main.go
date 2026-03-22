package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"ai-gateway/api"
	"ai-gateway/config"
	"ai-gateway/logger"
	"ai-gateway/metrics"
	"ai-gateway/model"
	"ai-gateway/pkg"
	"ai-gateway/provider"
	"ai-gateway/repository"
	"ai-gateway/router"
	"ai-gateway/scheduler"
	"ai-gateway/service"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect mysql: %v", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.APIKey{}, &model.UsageLog{}); err != nil {
		log.Fatalf("failed to auto migrate database: %v", err)
	}

	redisClient, err := pkg.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	usageRepo := repository.NewUsageRepository(db)

	appMetrics := metrics.NewCollector()

	authService := service.NewAuthService(userRepo, cfg.Auth)
	apiKeyService := service.NewAPIKeyService(apiKeyRepo)
	statsService := service.NewStatsService(redisClient)
	usageStatsService := service.NewUsageStatsService(usageRepo)
	
	providerRegistry := provider.NewRegistry(
		provider.NewOpenAIProvider(cfg.Providers.OpenAI.BaseURL, cfg.Providers.OpenAI.APIKeys),
		provider.NewDeepSeekProvider(cfg.Providers.DeepSeek.BaseURL, cfg.Providers.DeepSeek.APIKeys),
		provider.NewClaudeProvider(cfg.Providers.Claude.BaseURL, cfg.Providers.Claude.APIKeys),
	)
	providerScheduler := scheduler.New(providerRegistry, scheduler.Config{
		HealthCheckInterval: time.Duration(cfg.Scheduler.HealthCheckIntervalSec) * time.Second,
		HealthCheckTimeout:  time.Duration(cfg.Scheduler.HealthCheckTimeoutSec) * time.Second,
	}, appMetrics)

	backgroundCtx := context.Background()
	providerScheduler.Start(backgroundCtx)

	asyncLogger := logger.NewAsyncUsageLogger(usageRepo, logger.Config{
		QueueSize:     cfg.Logging.QueueSize,
		WorkerCount:   cfg.Logging.WorkerCount,
		BatchSize:     cfg.Logging.BatchSize,
		FlushInterval: time.Duration(cfg.Logging.FlushIntervalMS) * time.Millisecond,
	}, appMetrics)
	asyncLogger.Start(backgroundCtx)

	chatService := service.NewChatService(providerScheduler, asyncLogger, apiKeyRepo, statsService, appMetrics)
	authHandler := api.NewAuthHandler(authService)
	apiKeyHandler := api.NewAPIKeyHandler(apiKeyService)
	chatHandler := api.NewChatHandler(chatService)
	statsHandler := api.NewStatsHandler(usageStatsService, providerScheduler)
	engine := router.NewRouter(cfg, authService, apiKeyService, redisClient, appMetrics, authHandler, apiKeyHandler, chatHandler, statsHandler)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.App.Port),
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("AI Gateway started on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("failed to start server: %v", err)
	}
}

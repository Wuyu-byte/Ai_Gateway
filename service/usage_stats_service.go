package service

import (
	"context"

	"ai-gateway/repository"
)

type UsageStatsService struct {
	usageRepo *repository.UsageRepository
}

func NewUsageStatsService(usageRepo *repository.UsageRepository) *UsageStatsService {
	return &UsageStatsService{usageRepo: usageRepo}
}

func (s *UsageStatsService) DailyUsage(ctx context.Context, userID *uint, days int) ([]repository.DailyUsageStat, error) {
	if days <= 0 {
		days = 7
	}
	return s.usageRepo.DailyUsage(ctx, userID, days)
}

func (s *UsageStatsService) UserUsage(ctx context.Context, userID uint) ([]repository.UserUsageStat, error) {
	return s.usageRepo.UserUsage(ctx, userID)
}

func (s *UsageStatsService) ProviderUsage(ctx context.Context, userID uint) ([]repository.ProviderUsageStat, error) {
	return s.usageRepo.ProviderUsage(ctx, userID)
}

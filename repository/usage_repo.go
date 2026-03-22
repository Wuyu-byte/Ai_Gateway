package repository

import (
	"context"

	"ai-gateway/model"

	"gorm.io/gorm"
)

type UsageRepository struct {
	db *gorm.DB
}

func NewUsageRepository(db *gorm.DB) *UsageRepository {
	return &UsageRepository{db: db}
}

func (r *UsageRepository) Create(ctx context.Context, usage *model.UsageLog) error {
	return r.db.WithContext(ctx).Create(usage).Error
}

func (r *UsageRepository) BatchCreate(ctx context.Context, usages []*model.UsageLog) error {
	if len(usages) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).CreateInBatches(usages, len(usages)).Error
}

type DailyUsageStat struct {
	Date           string `json:"date"`
	RequestTokens  int64  `json:"request_tokens"`
	ResponseTokens int64  `json:"response_tokens"`
	TotalTokens    int64  `json:"total_tokens"`
	RequestCount   int64  `json:"request_count"`
}

type UserUsageStat struct {
	UserID         uint  `json:"user_id"`
	RequestTokens  int64 `json:"request_tokens"`
	ResponseTokens int64 `json:"response_tokens"`
	TotalTokens    int64 `json:"total_tokens"`
	RequestCount   int64 `json:"request_count"`
}

type ProviderUsageStat struct {
	Provider       string `json:"provider"`
	RequestTokens  int64  `json:"request_tokens"`
	ResponseTokens int64  `json:"response_tokens"`
	TotalTokens    int64  `json:"total_tokens"`
	RequestCount   int64  `json:"request_count"`
}

func (r *UsageRepository) DailyUsage(ctx context.Context, userID *uint, days int) ([]DailyUsageStat, error) {
	var result []DailyUsageStat
	query := r.db.WithContext(ctx).Model(&model.UsageLog{}).
		Select("DATE(created_at) AS date, SUM(request_tokens) AS request_tokens, SUM(response_tokens) AS response_tokens, SUM(request_tokens + response_tokens) AS total_tokens, COUNT(*) AS request_count")
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	err := query.Group("DATE(created_at)").
		Order("DATE(created_at) DESC").
		Limit(days).
		Scan(&result).Error
	return result, err
}

func (r *UsageRepository) UserUsage(ctx context.Context, userID uint) ([]UserUsageStat, error) {
	var result []UserUsageStat
	err := r.db.WithContext(ctx).Model(&model.UsageLog{}).
		Select("user_id, SUM(request_tokens) AS request_tokens, SUM(response_tokens) AS response_tokens, SUM(request_tokens + response_tokens) AS total_tokens, COUNT(*) AS request_count").
		Where("user_id = ?", userID).
		Group("user_id").
		Order("total_tokens DESC").
		Scan(&result).Error
	return result, err
}

func (r *UsageRepository) ProviderUsage(ctx context.Context, userID uint) ([]ProviderUsageStat, error) {
	var result []ProviderUsageStat
	err := r.db.WithContext(ctx).Model(&model.UsageLog{}).
		Select("provider, SUM(request_tokens) AS request_tokens, SUM(response_tokens) AS response_tokens, SUM(request_tokens + response_tokens) AS total_tokens, COUNT(*) AS request_count").
		Where("user_id = ?", userID).
		Group("provider").
		Order("total_tokens DESC").
		Scan(&result).Error
	return result, err
}

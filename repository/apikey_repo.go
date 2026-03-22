package repository

import (
	"context"
	"time"

	"ai-gateway/model"

	"gorm.io/gorm"
)

type APIKeyRepository struct {
	db *gorm.DB
}

func NewAPIKeyRepository(db *gorm.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, apiKey *model.APIKey) error {
	return r.db.WithContext(ctx).Create(apiKey).Error
}

func (r *APIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	var apiKey model.APIKey
	if err := r.db.WithContext(ctx).
		Preload("User"). //预加载关联的User数据
		Where("key_hash = ?", keyHash).
		First(&apiKey).Error; err != nil {
		return nil, err
	}

	return &apiKey, nil
}

func (r *APIKeyRepository) ListByUserID(ctx context.Context, userID uint) ([]model.APIKey, error) {
	var apiKeys []model.APIKey
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&apiKeys).Error; err != nil {
		return nil, err
	}

	return apiKeys, nil
}

func (r *APIKeyRepository) GetByIDAndUserID(ctx context.Context, id, userID uint) (*model.APIKey, error) {
	var apiKey model.APIKey
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&apiKey).Error; err != nil {
		return nil, err
	}

	return &apiKey, nil
}

func (r *APIKeyRepository) UpdateStatus(ctx context.Context, id, userID uint, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.APIKey{}). //指定操作的表
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

func (r *APIKeyRepository) TouchLastUsed(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.APIKey{}).
		Where("id = ?", id).
		Update("last_used_at", &now).Error
}

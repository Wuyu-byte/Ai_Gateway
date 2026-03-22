package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"ai-gateway/model"
	"ai-gateway/repository"

	"gorm.io/gorm"
)

var (
	ErrInvalidAPIKey = errors.New("invalid api key")
	ErrAPIKeyBlocked = errors.New("api key is disabled")
)

type APIKeyService struct {
	apiKeyRepo *repository.APIKeyRepository
}

type CreatedAPIKey struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Key        string `json:"key"`
	KeyPreview string `json:"key_preview"`
	Status     string `json:"status"`
	RateLimit  int    `json:"rate_limit"`
}

func NewAPIKeyService(apiKeyRepo *repository.APIKeyRepository) *APIKeyService {
	return &APIKeyService{apiKeyRepo: apiKeyRepo}
}

func (s *APIKeyService) Create(ctx context.Context, userID uint, name string, rateLimit int) (*CreatedAPIKey, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}
	if rateLimit <= 0 {
		rateLimit = 60
	}

	rawKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}
	keyHash := hashAPIKey(rawKey)

	apiKey := &model.APIKey{
		UserID:     userID,
		Name:       name,
		LegacyHash: keyHash,
		KeyHash:    keyHash,
		KeyPreview: maskAPIKey(rawKey),
		Status:     "active",
		RateLimit:  rateLimit,
	}
	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, err
	}

	return &CreatedAPIKey{
		ID:         apiKey.ID,
		Name:       apiKey.Name,
		Key:        rawKey,
		KeyPreview: apiKey.KeyPreview,
		Status:     apiKey.Status,
		RateLimit:  apiKey.RateLimit,
	}, nil
}

func (s *APIKeyService) List(ctx context.Context, userID uint) ([]model.APIKey, error) {
	return s.apiKeyRepo.ListByUserID(ctx, userID)
}

func (s *APIKeyService) Disable(ctx context.Context, userID, apiKeyID uint) error {
	if _, err := s.apiKeyRepo.GetByIDAndUserID(ctx, apiKeyID, userID); err != nil {
		return err
	}

	return s.apiKeyRepo.UpdateStatus(ctx, apiKeyID, userID, "disabled")
}

func (s *APIKeyService) Authenticate(ctx context.Context, rawKey string) (*model.APIKey, *model.User, error) {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return nil, nil, ErrInvalidAPIKey
	}

	computedHash := hashAPIKey(rawKey)
	apiKey, err := s.apiKeyRepo.GetByHash(ctx, computedHash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrInvalidAPIKey
		}
		return nil, nil, err
	}

	if subtle.ConstantTimeCompare([]byte(computedHash), []byte(apiKey.KeyHash)) != 1 {
		return nil, nil, ErrInvalidAPIKey
	}

	if apiKey.Status != "active" || apiKey.User.Status != "active" {
		return nil, nil, ErrAPIKeyBlocked
	}

	return apiKey, &apiKey.User, nil
}

func hashAPIKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

func generateAPIKey() (string, error) {
	entropy := make([]byte, 24)
	if _, err := rand.Read(entropy); err != nil {
		return "", err
	}

	return fmt.Sprintf("sk-%s", hex.EncodeToString(entropy)), nil
}

func maskAPIKey(rawKey string) string {
	if len(rawKey) <= 10 {
		return rawKey
	}

	return rawKey[:6] + "..." + rawKey[len(rawKey)-4:]
}

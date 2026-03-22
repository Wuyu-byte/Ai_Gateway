package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type StatsService struct {
	redisClient *redis.Client
}

func NewStatsService(redisClient *redis.Client) *StatsService {
	return &StatsService{redisClient: redisClient}
}

func (s *StatsService) RecordRequest(ctx context.Context, apiKeyID, userID uint, providerName string) error {
	dateKey := time.Now().Format("20060102")
	pipe := s.redisClient.Pipeline()

	requestKey := fmt.Sprintf("usage:requests:%s", dateKey)
	userRequestKey := fmt.Sprintf("usage:user:requests:%d:%s", userID, dateKey)
	apiKeyRequestKey := fmt.Sprintf("usage:apikey:requests:%d:%s", apiKeyID, dateKey)
	providerRequestKey := fmt.Sprintf("usage:provider:requests:%s:%s", providerName, dateKey)

	pipe.Incr(ctx, requestKey)
	pipe.Incr(ctx, userRequestKey)
	pipe.Incr(ctx, apiKeyRequestKey)
	pipe.Incr(ctx, providerRequestKey)
	pipe.Expire(ctx, requestKey, 7*24*time.Hour)
	pipe.Expire(ctx, userRequestKey, 7*24*time.Hour)
	pipe.Expire(ctx, apiKeyRequestKey, 7*24*time.Hour)
	pipe.Expire(ctx, providerRequestKey, 7*24*time.Hour)

	_, err := pipe.Exec(ctx)
	return err
}

func (s *StatsService) RecordTokens(ctx context.Context, apiKeyID, userID uint, providerName string, requestTokens, responseTokens int) error {
	dateKey := time.Now().Format("20060102")
	pipe := s.redisClient.Pipeline()

	pipe.IncrBy(ctx, fmt.Sprintf("usage:tokens:prompt:%s", dateKey), int64(requestTokens))
	pipe.IncrBy(ctx, fmt.Sprintf("usage:tokens:completion:%s", dateKey), int64(responseTokens))
	pipe.IncrBy(ctx, fmt.Sprintf("usage:user:tokens:%d:%s", userID, dateKey), int64(requestTokens+responseTokens))
	pipe.IncrBy(ctx, fmt.Sprintf("usage:apikey:tokens:%d:%s", apiKeyID, dateKey), int64(requestTokens+responseTokens))
	pipe.IncrBy(ctx, fmt.Sprintf("usage:provider:tokens:%s:%s", providerName, dateKey), int64(requestTokens+responseTokens))
	pipe.Expire(ctx, fmt.Sprintf("usage:tokens:prompt:%s", dateKey), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("usage:tokens:completion:%s", dateKey), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("usage:user:tokens:%d:%s", userID, dateKey), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("usage:apikey:tokens:%d:%s", apiKeyID, dateKey), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("usage:provider:tokens:%s:%s", providerName, dateKey), 7*24*time.Hour)

	_, err := pipe.Exec(ctx)
	return err
}

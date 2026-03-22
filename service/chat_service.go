package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"ai-gateway/logger"
	"ai-gateway/metrics"
	"ai-gateway/model"
	"ai-gateway/provider"
	"ai-gateway/repository"
	"ai-gateway/scheduler"
)

type ChatService struct {
	scheduler   *scheduler.Scheduler
	asyncLogger *logger.AsyncUsageLogger
	apiKeyRepo  *repository.APIKeyRepository
	statsSvc    *StatsService
	metrics     *metrics.Collector
	defaultCode int
}

func NewChatService(
	scheduler *scheduler.Scheduler,
	asyncLogger *logger.AsyncUsageLogger,
	apiKeyRepo *repository.APIKeyRepository,
	statsSvc *StatsService,
	collector *metrics.Collector,
) *ChatService {
	return &ChatService{
		scheduler:   scheduler,
		asyncLogger: asyncLogger,
		apiKeyRepo:  apiKeyRepo,
		statsSvc:    statsSvc,
		metrics:     collector,
		defaultCode: 200,
	}
}

func (s *ChatService) Chat(
	ctx context.Context,
	user *model.User,
	apiKey *model.APIKey,
	req *provider.ChatRequest,
) (*provider.ChatResponse, error) {
	return s.execute(ctx, user, apiKey, req, nil)
}

func (s *ChatService) StreamChat(
	ctx context.Context,
	user *model.User,
	apiKey *model.APIKey,
	req *provider.ChatRequest,
	send provider.StreamSender,
) (*provider.ChatResponse, error) {
	return s.execute(ctx, user, apiKey, req, send)
}

func (s *ChatService) execute(
	ctx context.Context,
	user *model.User,
	apiKey *model.APIKey,
	req *provider.ChatRequest,
	send provider.StreamSender,
) (*provider.ChatResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return nil, errors.New("model is required")
	}

	if len(req.Messages) == 0 {
		return nil, errors.New("messages is required")
	}

	selectedProvider, err := s.scheduler.Select(req.Model)
	if err != nil {
		return nil, err
	}

	if s.metrics != nil {
		s.metrics.ObserveProviderCall(selectedProvider.Name(), req.Model, req.Stream)
	}

	start := time.Now()
	var resp *provider.ChatResponse
	if req.Stream {
		resp, err = selectedProvider.StreamChat(ctx, req, send)
	} else {
		resp, err = selectedProvider.Chat(ctx, req)
	}
	latency := time.Since(start)
	recordSuccess := err == nil || errors.Is(err, provider.ErrStreamingUnsupported)
	s.scheduler.RecordResult(selectedProvider.Name(), latency, recordSuccess)

	statusCode := s.defaultCode
	errorMessage := ""
	requestTokens := estimateRequestTokens(req)
	responseTokens := 0

	if err != nil {
		statusCode = 502
		errorMessage = err.Error()
	} else if resp != nil {
		if resp.Usage.RequestTokens > 0 {
			requestTokens = resp.Usage.RequestTokens
		}
		if resp.Usage.ResponseTokens > 0 {
			responseTokens = resp.Usage.ResponseTokens
		} else {
			responseTokens = estimateTextTokens(resp.FirstMessageContent())
		}
	}

	usage := &model.UsageLog{
		UserID:         user.ID,
		APIKeyID:       apiKey.ID,
		Provider:       selectedProvider.Name(),
		Model:          req.Model,
		RequestTokens:  requestTokens,
		ResponseTokens: responseTokens,
		LatencyMS:      latency.Milliseconds(),
		StatusCode:     statusCode,
		ErrorMessage:   errorMessage,
	}

	if s.asyncLogger != nil {
		_ = s.asyncLogger.Enqueue(usage)
	}
	_ = s.apiKeyRepo.TouchLastUsed(ctx, apiKey.ID)
	if s.statsSvc != nil {
		_ = s.statsSvc.RecordRequest(ctx, apiKey.ID, user.ID, selectedProvider.Name())
		_ = s.statsSvc.RecordTokens(ctx, apiKey.ID, user.ID, selectedProvider.Name(), requestTokens, responseTokens)
	}

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func estimateRequestTokens(req *provider.ChatRequest) int {
	total := len(req.Model)
	for _, message := range req.Messages {
		total += len(message.Role) + len(message.Content)
	}

	return roughTokenCount(total)
}

func estimateTextTokens(content string) int {
	return roughTokenCount(len(content))
}

func roughTokenCount(chars int) int {
	if chars <= 0 {
		return 0
	}

	count := chars / 4
	if count == 0 {
		return 1
	}

	return count
}

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrModelNotSupported    = errors.New("model is not supported")
	ErrStreamingUnsupported = errors.New("streaming is not supported by provider")
)

type StreamEvent struct {
	Data string
}

type StreamSender func(event StreamEvent) error

type HealthStatus struct {
	Healthy      bool
	Latency      time.Duration
	CheckedAt    time.Time
	ErrorMessage string
}

type Provider interface {
	Name() string
	SupportsModel(model string) bool
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	StreamChat(ctx context.Context, req *ChatRequest, send StreamSender) (*ChatResponse, error)
	HealthCheck(ctx context.Context) (*HealthStatus, error)
}

type Registry struct {
	providers []Provider
}

func NewRegistry(providers ...Provider) *Registry {
	filtered := make([]Provider, 0, len(providers))
	for _, item := range providers {
		if item != nil {
			filtered = append(filtered, item)
		}
	}

	return &Registry{providers: filtered}
}

func (r *Registry) Resolve(model string) (Provider, error) {
	model = strings.TrimSpace(model)
	for _, item := range r.providers {
		if item.SupportsModel(model) {
			return item, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrModelNotSupported, model)
}

func (r *Registry) ProvidersForModel(model string) []Provider {
	model = strings.TrimSpace(model)
	providers := make([]Provider, 0, len(r.providers))
	for _, item := range r.providers {
		if item.SupportsModel(model) {
			providers = append(providers, item)
		}
	}

	return providers
}

func (r *Registry) All() []Provider {
	result := make([]Provider, 0, len(r.providers))
	result = append(result, r.providers...)
	return result
}

type ChatRequest struct {
	Model       string        `json:"model" binding:"required"`
	Messages    []ChatMessage `json:"messages" binding:"required"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role" binding:"required"`
	Content string `json:"content" binding:"required"`
}

type ChatResponse struct {
	ID       string       `json:"id"`
	Object   string       `json:"object,omitempty"`
	Created  int64        `json:"created,omitempty"`
	Model    string       `json:"model"`
	Provider string       `json:"provider"`
	Choices  []ChatChoice `json:"choices"`
	Usage    Usage        `json:"usage"`
}

type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

type Usage struct {
	RequestTokens  int `json:"prompt_tokens"`
	ResponseTokens int `json:"completion_tokens"`
	TotalTokens    int `json:"total_tokens"`
}

func (r *ChatResponse) FirstMessageContent() string {
	if r == nil || len(r.Choices) == 0 {
		return ""
	}

	return r.Choices[0].Message.Content
}

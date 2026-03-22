package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type ClaudeProvider struct {
	name    string
	baseURL string
	apiKeys []string
	client  *http.Client
	counter uint64
}

func NewClaudeProvider(baseURL string, apiKeys []string) *ClaudeProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	return &ClaudeProvider{
		name:    "claude",
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKeys: apiKeys,
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *ClaudeProvider) Name() string {
	return p.name
}

func (p *ClaudeProvider) SupportsModel(model string) bool {
	return strings.HasPrefix(model, "claude")
}

func (p *ClaudeProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if len(p.apiKeys) == 0 {
		return nil, errors.New("claude provider has no configured api keys")
	}

	payload := map[string]any{
		"model":      req.Model,
		"messages":   toClaudeMessages(req.Messages),
		"max_tokens": 1024,
	}
	if req.MaxTokens != nil {
		payload["max_tokens"] = *req.MaxTokens
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/messages", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("x-api-key", p.nextKey())
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("claude api error: status=%d body=%s", httpResp.StatusCode, string(respBody))
	}

	var raw struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, err
	}

	content := ""
	for _, item := range raw.Content {
		if item.Type == "text" {
			content += item.Text
		}
	}

	return &ChatResponse{
		ID:       raw.ID,
		Object:   "chat.completion",
		Created:  time.Now().Unix(),
		Model:    raw.Model,
		Provider: p.name,
		Choices: []ChatChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			RequestTokens:  raw.Usage.InputTokens,
			ResponseTokens: raw.Usage.OutputTokens,
			TotalTokens:    raw.Usage.InputTokens + raw.Usage.OutputTokens,
		},
	}, nil
}

func (p *ClaudeProvider) StreamChat(ctx context.Context, req *ChatRequest, send StreamSender) (*ChatResponse, error) {
	return nil, ErrStreamingUnsupported
}

func (p *ClaudeProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if len(p.apiKeys) == 0 {
		return &HealthStatus{
			Healthy:      false,
			CheckedAt:    time.Now(),
			ErrorMessage: "claude provider has no configured api keys",
		}, errors.New("claude provider has no configured api keys")
	}

	start := time.Now()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", p.nextKey())
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	httpResp, err := p.client.Do(httpReq)
	latency := time.Since(start)
	if err != nil {
		return &HealthStatus{
			Healthy:      false,
			Latency:      latency,
			CheckedAt:    time.Now(),
			ErrorMessage: err.Error(),
		}, err
	}
	defer httpResp.Body.Close()
	_, _ = io.Copy(io.Discard, httpResp.Body)

	healthy := httpResp.StatusCode < http.StatusInternalServerError
	status := &HealthStatus{
		Healthy:   healthy,
		Latency:   latency,
		CheckedAt: time.Now(),
	}
	if !healthy {
		status.ErrorMessage = fmt.Sprintf("health check returned status %d", httpResp.StatusCode)
		return status, errors.New(status.ErrorMessage)
	}

	return status, nil
}

func (p *ClaudeProvider) nextKey() string {
	index := atomic.AddUint64(&p.counter, 1)
	return p.apiKeys[(index-1)%uint64(len(p.apiKeys))]
}

func toClaudeMessages(messages []ChatMessage) []map[string]string {
	result := make([]map[string]string, 0, len(messages))
	for _, message := range messages {
		role := message.Role
		if role == "system" {
			role = "user"
		}
		result = append(result, map[string]string{
			"role":    role,
			"content": message.Content,
		})
	}

	return result
}

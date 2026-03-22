package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type DeepSeekProvider struct {
	name    string
	baseURL string
	apiKeys []string
	client  *http.Client
	counter uint64
}

func NewDeepSeekProvider(baseURL string, apiKeys []string) *DeepSeekProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.deepseek.com/v1"
	}

	return &DeepSeekProvider{
		name:    "deepseek",
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKeys: apiKeys,
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *DeepSeekProvider) Name() string {
	return p.name
}

func (p *DeepSeekProvider) SupportsModel(model string) bool {
	return strings.HasPrefix(model, "deepseek")
}

func (p *DeepSeekProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if len(p.apiKeys) == 0 {
		return nil, errors.New("deepseek provider has no configured api keys")
	}

	httpReq, err := newJSONRequest(ctx, http.MethodPost, p.baseURL+"/chat/completions", buildOpenAICompatiblePayload(req, false))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.nextKey())

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
		return nil, fmt.Errorf("deepseek api error: status=%d body=%s", httpResp.StatusCode, string(respBody))
	}

	return decodeOpenAICompatibleResponse(p.name, respBody)
}

func (p *DeepSeekProvider) StreamChat(ctx context.Context, req *ChatRequest, send StreamSender) (*ChatResponse, error) {
	if len(p.apiKeys) == 0 {
		return nil, errors.New("deepseek provider has no configured api keys")
	}

	httpReq, err := newJSONRequest(ctx, http.MethodPost, p.baseURL+"/chat/completions", buildOpenAICompatiblePayload(req, true))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.nextKey())
	httpReq.Header.Set("Accept", "text/event-stream")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("deepseek stream api error: status=%d body=%s", httpResp.StatusCode, string(respBody))
	}

	return streamOpenAICompatibleResponse(httpResp, p.name, req.Model, send)
}

func (p *DeepSeekProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if len(p.apiKeys) == 0 {
		return &HealthStatus{
			Healthy:      false,
			CheckedAt:    time.Now(),
			ErrorMessage: "deepseek provider has no configured api keys",
		}, errors.New("deepseek provider has no configured api keys")
	}

	return probeOpenAICompatibleHealth(ctx, p.client, p.baseURL, p.nextKey())
}

func (p *DeepSeekProvider) nextKey() string {
	index := atomic.AddUint64(&p.counter, 1)
	return p.apiKeys[(index-1)%uint64(len(p.apiKeys))]
}

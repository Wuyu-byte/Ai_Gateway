package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func buildOpenAICompatiblePayload(req *ChatRequest, stream bool) map[string]any {
	payload := map[string]any{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   stream,
	}

	if req.Temperature != nil {
		payload["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		payload["max_tokens"] = *req.MaxTokens
	}
	if stream {
		payload["stream_options"] = map[string]bool{
			"include_usage": true,
		}
	}

	return payload
}

func newJSONRequest(ctx context.Context, method, url string, payload any) (*http.Request, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func decodeOpenAICompatibleResponse(providerName string, body []byte) (*ChatResponse, error) {
	var raw struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	choices := make([]ChatChoice, 0, len(raw.Choices))
	for _, item := range raw.Choices {
		choices = append(choices, ChatChoice{
			Index: item.Index,
			Message: ChatMessage{
				Role:    item.Message.Role,
				Content: item.Message.Content,
			},
			FinishReason: item.FinishReason,
		})
	}

	return &ChatResponse{
		ID:       raw.ID,
		Object:   raw.Object,
		Created:  raw.Created,
		Model:    raw.Model,
		Provider: providerName,
		Choices:  choices,
		Usage: Usage{
			RequestTokens:  raw.Usage.PromptTokens,
			ResponseTokens: raw.Usage.CompletionTokens,
			TotalTokens:    raw.Usage.TotalTokens,
		},
	}, nil
}

func streamOpenAICompatibleResponse(
	resp *http.Response,
	providerName string,
	reqModel string,
	send StreamSender,
) (*ChatResponse, error) {
	reader := bufio.NewReader(resp.Body)
	contentBuilder := strings.Builder{}
	response := &ChatResponse{
		Object:   "chat.completion.chunk",
		Created:  time.Now().Unix(),
		Model:    reqModel,
		Provider: providerName,
		Choices: []ChatChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role: "assistant",
				},
			},
		},
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "data:") {
			payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
			if payload != "" {
				if err := send(StreamEvent{Data: payload}); err != nil {
					return nil, err
				}

				if payload == "[DONE]" {
					break
				}

				var raw struct {
					ID      string `json:"id"`
					Object  string `json:"object"`
					Created int64  `json:"created"`
					Model   string `json:"model"`
					Choices []struct {
						Index int `json:"index"`
						Delta struct {
							Role    string `json:"role"`
							Content string `json:"content"`
						} `json:"delta"`
						FinishReason string `json:"finish_reason"`
					} `json:"choices"`
					Usage *struct {
						PromptTokens     int `json:"prompt_tokens"`
						CompletionTokens int `json:"completion_tokens"`
						TotalTokens      int `json:"total_tokens"`
					} `json:"usage"`
				}

				if err := json.Unmarshal([]byte(payload), &raw); err == nil {
					if raw.ID != "" {
						response.ID = raw.ID
					}
					if raw.Object != "" {
						response.Object = raw.Object
					}
					if raw.Created > 0 {
						response.Created = raw.Created
					}
					if raw.Model != "" {
						response.Model = raw.Model
					}
					for _, choice := range raw.Choices {
						if choice.Delta.Role != "" {
							response.Choices[0].Message.Role = choice.Delta.Role
						}
						if choice.Delta.Content != "" {
							contentBuilder.WriteString(choice.Delta.Content)
						}
						if choice.FinishReason != "" {
							response.Choices[0].FinishReason = choice.FinishReason
						}
					}
					if raw.Usage != nil {
						response.Usage = Usage{
							RequestTokens:  raw.Usage.PromptTokens,
							ResponseTokens: raw.Usage.CompletionTokens,
							TotalTokens:    raw.Usage.TotalTokens,
						}
					}
				}
			}
		}

		if err == io.EOF {
			break
		}
	}

	response.Choices[0].Message.Content = contentBuilder.String()
	return response, nil
}

func probeOpenAICompatibleHealth(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	apiKey string,
) (*HealthStatus, error) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return &HealthStatus{
			Healthy:      false,
			Latency:      latency,
			CheckedAt:    time.Now(),
			ErrorMessage: err.Error(),
		}, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	healthy := resp.StatusCode < http.StatusInternalServerError
	status := &HealthStatus{
		Healthy:   healthy,
		Latency:   latency,
		CheckedAt: time.Now(),
	}
	if !healthy {
		status.ErrorMessage = fmt.Sprintf("health check returned status %d", resp.StatusCode)
		return status, fmt.Errorf(status.ErrorMessage)
	}

	return status, nil
}

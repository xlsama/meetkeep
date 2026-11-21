// Package llm talks to a service that speaks OpenAI's chat completions.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrNotConfigured = errors.New("纪要服务还没有配置")

type Client struct {
	BaseURL string
	Key     string
	Model   string
	HTTP    *http.Client
}

func New(baseURL, key, model string) (*Client, error) {
	if baseURL == "" || key == "" || model == "" {
		return nil, ErrNotConfigured
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Key:     key,
		Model:   model,
		HTTP:    &http.Client{Timeout: 5 * time.Minute},
	}, nil
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model          string    `json:"model"`
	Messages       []Message `json:"messages"`
	MaxTokens      int       `json:"max_tokens,omitempty"`
	Temperature    *float64  `json:"temperature,omitempty"`
	ResponseFormat *format   `json:"response_format,omitempty"`
}

type format struct {
	Type string `json:"type"`
}

// Complete returns the model's answer to messages.
func (c *Client) Complete(ctx context.Context, messages []Message, maxTokens int) (string, error) {
	return c.send(ctx, request{Model: c.Model, Messages: messages, MaxTokens: maxTokens})
}

// CompleteJSON returns the model's answer to user under the instructions of
// system, which ask for a JSON object.
func (c *Client) CompleteJSON(ctx context.Context, system, user string) (string, error) {
	temperature := 0.2
	return c.send(ctx, request{
		Model:          c.Model,
		Messages:       []Message{{Role: "system", Content: system}, {Role: "user", Content: user}},
		Temperature:    &temperature,
		ResponseFormat: &format{Type: "json_object"},
	})
}

func (c *Client) send(ctx context.Context, r request) (string, error) {
	body, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Key)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		text := []rune(strings.TrimSpace(string(data)))
		return "", fmt.Errorf("%s 返回 %s：%s", c.Model, res.Status, string(text[:min(len(text), 160)]))
	}
	var result struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", errors.New("模型没有返回内容")
	}
	return result.Choices[0].Message.Content, nil
}

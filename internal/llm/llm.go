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
	"sync"
	"time"
)

var ErrNotConfigured = errors.New("纪要服务还没有配置")

type Client struct {
	BaseURL string
	Key     string
	Model   string
	HTTP    *http.Client

	mu   sync.Mutex
	used Usage
	// thinks tells that the service refused a request not to think.
	thinks bool
}

// Usage is what a client asked of the model so far.
type Usage struct {
	Requests         int
	PromptTokens     int
	CompletionTokens int
}

func (c *Client) Used() Usage {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.used
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
	Thinking       *thinking `json:"thinking,omitempty"`
}

type thinking struct {
	Type string `json:"type"`
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

// CompleteQuick is CompleteJSON for work that needs no reasoning: it asks
// the model not to think before it answers, which makes a model that
// would several times faster. A service that does not know that request
// is asked again without it.
func (c *Client) CompleteQuick(ctx context.Context, system, user string) (string, error) {
	temperature := 0.2
	r := request{
		Model:          c.Model,
		Messages:       []Message{{Role: "system", Content: system}, {Role: "user", Content: user}},
		Temperature:    &temperature,
		ResponseFormat: &format{Type: "json_object"},
	}
	c.mu.Lock()
	thinks := c.thinks
	c.mu.Unlock()
	if thinks {
		return c.send(ctx, r)
	}
	r.Thinking = &thinking{Type: "disabled"}
	answer, err := c.send(ctx, r)
	var refused *refusal
	if !errors.As(err, &refused) {
		return answer, err
	}
	c.mu.Lock()
	c.thinks = true
	c.mu.Unlock()
	r.Thinking = nil
	return c.send(ctx, r)
}

// refusal is the service's answer to a request it does not accept.
type refusal struct{ message string }

func (r *refusal) Error() string { return r.message }

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
		message := fmt.Sprintf("%s 返回 %s：%s", c.Model, res.Status, string(text[:min(len(text), 160)]))
		if res.StatusCode == http.StatusBadRequest || res.StatusCode == http.StatusUnprocessableEntity {
			return "", &refusal{message}
		}
		return "", errors.New(message)
	}
	var result struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	c.mu.Lock()
	c.used.Requests++
	c.used.PromptTokens += result.Usage.PromptTokens
	c.used.CompletionTokens += result.Usage.CompletionTokens
	c.mu.Unlock()
	if len(result.Choices) == 0 {
		return "", errors.New("模型没有返回内容")
	}
	return result.Choices[0].Message.Content, nil
}

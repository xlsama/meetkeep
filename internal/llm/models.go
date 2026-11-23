package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"

	"github.com/xlsama/meetkeep/internal/config"
)

const (
	deepSeekURL   = "https://api.deepseek.com"
	openRouterURL = "https://openrouter.ai/api/v1"
)

// Models lists the models the provider of the settings offers, in the order
// the provider gives them.
func Models(ctx context.Context, s config.LLMSettings) ([]string, error) {
	switch s.Provider {
	case config.LLMAnthropic:
		return claudeModels(ctx, s.Anthropic.Key)
	case config.LLMDeepSeek:
		if s.DeepSeek.Key == "" {
			return nil, ErrNotConfigured
		}
		return listModels(ctx, http.DefaultClient, deepSeekURL, s.DeepSeek.Key)
	case config.LLMOpenRouter:
		return listModels(ctx, http.DefaultClient, openRouterURL, s.OpenRouter.Key)
	}
	chosen := config.Chosen(s.Services, s.Service)
	return listModels(ctx, http.DefaultClient, chosen.BaseURL, chosen.Key)
}

func claudeModels(ctx context.Context, key string) ([]string, error) {
	if key == "" {
		return nil, ErrNotConfigured
	}
	client := anthropic.NewClient(option.WithAPIKey(key))
	ids := []string{}
	pager := client.Models.ListAutoPaging(ctx, anthropic.ModelListParams{
		Limit:     anthropic.Int(1000),
		Lifecycle: []string{"active"},
	})
	for pager.Next() {
		ids = append(ids, pager.Current().ID)
	}
	if err := pager.Err(); err != nil {
		var failed *anthropic.Error
		if errors.As(err, &failed) {
			return nil, fmt.Errorf("模型列表返回 %d：%s", failed.StatusCode, failed.Message)
		}
		return nil, err
	}
	return ids, nil
}

// listModels asks a service that speaks OpenAI's API what models it has. A
// service that lists them to anyone needs no key.
func listModels(ctx context.Context, client *http.Client, baseURL, key string) ([]string, error) {
	if baseURL == "" {
		return nil, ErrNotConfigured
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/models", nil)
	if err != nil {
		return nil, err
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		text := []rune(strings.TrimSpace(string(data)))
		return nil, fmt.Errorf("模型列表返回 %s：%s", res.Status, string(text[:min(len(text), 160)]))
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(result.Data))
	for _, model := range result.Data {
		if model.ID != "" {
			ids = append(ids, model.ID)
		}
	}
	return ids, nil
}

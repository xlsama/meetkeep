package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/v2"
)

const claudeMaxTokens = 32000

func (c *Client) sendClaude(ctx context.Context, system, user string, maxTokens int, object bool) (string, error) {
	if maxTokens == 0 {
		maxTokens = claudeMaxTokens
	}
	params := anthropic.BetaMessageNewParams{
		Model:        c.Model,
		MaxTokens:    int64(maxTokens),
		Betas:        []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		Fallbacks:    anthropic.BetaFallbacksParamOfDefault(),
		OutputConfig: anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffortHigh},
	}
	if system != "" {
		params.System = []anthropic.BetaTextBlockParam{{Text: system, CacheControl: anthropic.NewBetaCacheControlEphemeralParam()}}
	}
	params.Messages = []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(user))}

	stream := c.claude.Beta.Messages.NewStreaming(ctx, params)
	var message anthropic.BetaMessage
	for stream.Next() {
		if err := message.Accumulate(stream.Current()); err != nil {
			return "", err
		}
	}
	if err := stream.Err(); err != nil {
		var failed *anthropic.Error
		if errors.As(err, &failed) {
			return "", fmt.Errorf("%s 返回 %d：%s", c.Model, failed.StatusCode, failed.Message)
		}
		return "", err
	}

	c.mu.Lock()
	c.used.Requests++
	c.used.PromptTokens += int(message.Usage.InputTokens + message.Usage.CacheReadInputTokens + message.Usage.CacheCreationInputTokens)
	c.used.CachedTokens += int(message.Usage.CacheReadInputTokens)
	c.used.CompletionTokens += int(message.Usage.OutputTokens)
	c.mu.Unlock()

	if message.StopReason == anthropic.BetaStopReasonRefusal {
		return "", fmt.Errorf("%s 拒绝了这次请求", c.Model)
	}
	var text strings.Builder
	for _, block := range message.Content {
		if block.Type == "text" {
			text.WriteString(block.AsText().Text)
		}
	}
	if !object {
		return text.String(), nil
	}
	if message.StopReason == anthropic.BetaStopReasonMaxTokens {
		return "", fmt.Errorf("%s 的回答超出了长度上限", c.Model)
	}
	answer := text.String()
	start, end := strings.Index(answer, "{"), strings.LastIndex(answer, "}")
	if start < 0 || end < start {
		return "", errors.New("模型没有返回 JSON")
	}
	return answer[start : end+1], nil
}

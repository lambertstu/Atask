package engine

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"agent-base/internal/llm"

	"github.com/sashabaranov/go-openai"
)

const (
	MAX_RECOVERY_ATTEMPTS = 3
	BACKOFF_BASE_DELAY    = 1.0
	BACKOFF_MAX_DELAY     = 30.0
)

const CONTINUATION_MESSAGE = "Output limit hit. Continue directly from where you stopped -- no recap, no repetition. Pick up mid-sentence if needed."

type RecoveryManagerImpl struct {
	client        llm.LLMClient
	model         string
	contextMgr    ContextManager
	promptBuilder PromptBuilder
}

func NewRecoveryManager(client llm.LLMClient, model string, contextMgr ContextManager, promptBuilder PromptBuilder) *RecoveryManagerImpl {
	return &RecoveryManagerImpl{
		client:        client,
		model:         model,
		contextMgr:    contextMgr,
		promptBuilder: promptBuilder,
	}
}

func backoffDelay(attempt int) float64 {
	delay := BACKOFF_BASE_DELAY * float64(int(1)<<uint(attempt))
	if delay > BACKOFF_MAX_DELAY {
		delay = BACKOFF_MAX_DELAY
	}
	jitter := rand.Float64()
	return delay + jitter
}

func (rm *RecoveryManagerImpl) CreateWithRecovery(ctx context.Context, req openai.ChatCompletionRequest, messages *[]openai.ChatCompletionMessage) (*openai.ChatCompletionResponse, error) {
	maxOutputRecoveryCount := 0

	for {
		*messages = rm.contextMgr.MicroCompact(*messages)

		if rm.contextMgr.EstimateTokens(*messages) > CONTEXT_THRESHOLD {
			fmt.Println("[auto_compact triggered]")
			*messages = rm.contextMgr.AutoCompact(*messages)
		}

		req.Messages = append([]openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: rm.promptBuilder.Build(),
			},
		}, *messages...)

		var resp openai.ChatCompletionResponse
		var lastErr error

		for attempt := 0; attempt <= MAX_RECOVERY_ATTEMPTS; attempt++ {
			var err error
			resp, err = rm.client.CreateCompletion(ctx, req)
			if err == nil {
				break
			}
			lastErr = err

			errStr := strings.ToLower(err.Error())

			if strings.Contains(errStr, "prompt") && strings.Contains(errStr, "long") {
				fmt.Printf("[Recovery] Prompt too long. Compacting... (attempt %d)\n", attempt+1)
				*messages = rm.contextMgr.AutoCompact(*messages)
				req.Messages = append([]openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: rm.promptBuilder.Build(),
					},
				}, *messages...)
				continue
			}

			if attempt < MAX_RECOVERY_ATTEMPTS {
				delay := backoffDelay(attempt)
				fmt.Printf("[Recovery] API error: %v. Retrying in %.1fs (attempt %d/%d)\n", err, delay, attempt+1, MAX_RECOVERY_ATTEMPTS)
				time.Sleep(time.Duration(delay * float64(time.Second)))
				continue
			}

			return nil, fmt.Errorf("API call failed after %d retries: %w", MAX_RECOVERY_ATTEMPTS, lastErr)
		}

		if lastErr != nil {
			return nil, fmt.Errorf("no response received: %w", lastErr)
		}

		if len(resp.Choices) > 0 && resp.Choices[0].FinishReason == "length" {
			maxOutputRecoveryCount++
			if maxOutputRecoveryCount <= MAX_RECOVERY_ATTEMPTS {
				fmt.Printf("[Recovery] max_tokens hit (%d/%d). Injecting continuation...\n", maxOutputRecoveryCount, MAX_RECOVERY_ATTEMPTS)
				*messages = append(*messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: CONTINUATION_MESSAGE,
				})
				req.Messages = append([]openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: rm.promptBuilder.Build(),
					},
				}, *messages...)
				continue
			}
			return nil, fmt.Errorf("max_tokens recovery exhausted (%d attempts)", MAX_RECOVERY_ATTEMPTS)
		}

		return &resp, nil
	}
}

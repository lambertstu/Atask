package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"agent-base/internal/llm"
	"agent-base/pkg/utils"

	"github.com/sashabaranov/go-openai"
)

const (
	KEEP_RECENT       = 3
	TRANSCRIPT_DIR    = ".transcripts"
	CONTEXT_THRESHOLD = 50000
)

var PRESERVE_RESULT_TOOLS = map[string]bool{
	"read_file": true,
}

type ContextManagerImpl struct {
	client    llm.LLMClient
	model     string
	workdir   string
	threshold int
}

func NewContextManager(client llm.LLMClient, model, workdir string, threshold int) *ContextManagerImpl {
	return &ContextManagerImpl{
		client:    client,
		model:     model,
		workdir:   workdir,
		threshold: threshold,
	}
}

func (cm *ContextManagerImpl) EstimateTokens(messages []openai.ChatCompletionMessage) int {
	return utils.EstimateTokens(messages)
}

func (cm *ContextManagerImpl) MicroCompact(messages []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	var toolResults []struct {
		msgIdx  int
		toolID  string
		content string
	}

	for msgIdx, msg := range messages {
		if msg.Role == openai.ChatMessageRoleTool {
			toolResults = append(toolResults, struct {
				msgIdx  int
				toolID  string
				content string
			}{
				msgIdx:  msgIdx,
				toolID:  msg.ToolCallID,
				content: msg.Content,
			})
		}
	}

	if len(toolResults) <= KEEP_RECENT {
		return messages
	}

	toolNameMap := make(map[string]string)
	for _, msg := range messages {
		for _, tc := range msg.ToolCalls {
			toolNameMap[tc.ID] = tc.Function.Name
		}
	}

	toClear := toolResults[:len(toolResults)-KEEP_RECENT]
	for _, result := range toClear {
		if len(result.content) <= 100 {
			continue
		}

		toolName := toolNameMap[result.toolID]
		if toolName == "" {
			toolName = "unknown"
		}

		if PRESERVE_RESULT_TOOLS[toolName] {
			continue
		}

		messages[result.msgIdx].Content = fmt.Sprintf("[Previous: used %s]", toolName)
	}

	return messages
}

func (cm *ContextManagerImpl) AutoCompact(messages []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	transcriptDir := filepath.Join(cm.workdir, TRANSCRIPT_DIR)
	os.MkdirAll(transcriptDir, 0755)

	transcriptPath := filepath.Join(transcriptDir, fmt.Sprintf("transcript_%d.jsonl", time.Now().Unix()))
	file, err := os.Create(transcriptPath)
	if err == nil {
		encoder := json.NewEncoder(file)
		for _, msg := range messages {
			encoder.Encode(msg)
		}
		file.Close()
		fmt.Printf("[transcript saved: %s]\n", transcriptPath)
	}

	conversationData, _ := json.Marshal(messages)
	conversationText := string(conversationData)
	if len(conversationText) > 80000 {
		conversationText = conversationText[len(conversationText)-80000:]
	}

	prompt := fmt.Sprintf("Summarize this conversation for continuity. Include:\n1) What was accomplished\n2) Current state\n3) Key decisions made\nBe concise but preserve critical details.\n\n%s", conversationText)

	resp, err := cm.client.CreateCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: cm.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	})

	var summary string
	if err != nil || len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		summary = "No summary generated."
	} else {
		summary = resp.Choices[0].Message.Content
	}

	return []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("[Conversation compressed. Transcript: %s]\n\n%s", transcriptPath, summary),
		},
	}
}

func (cm *ContextManagerImpl) SaveLargeOutput(toolName, output string) string {
	if len(output) <= 4000 {
		return output
	}

	transcriptDir := filepath.Join(cm.workdir, TRANSCRIPT_DIR)
	os.MkdirAll(transcriptDir, 0755)

	filename := fmt.Sprintf("output_%s_%d.txt", toolName, time.Now().UnixNano())
	filePath := filepath.Join(transcriptDir, filename)

	err := os.WriteFile(filePath, []byte(output), 0644)
	if err != nil {
		return fmt.Sprintf("[Output too large (%d bytes), failed to save: %v]\n\nPreview:\n%s...\n[End of preview]",
			len(output), err, utils.Truncate(output, 4000))
	}

	return fmt.Sprintf("[Output too large (%d bytes), saved to %s]\n\nPreview:\n%s...\n[End of preview]",
		len(output), filePath, utils.Truncate(output, 4000))
}

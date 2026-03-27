package openrouter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const baseURL = "https://openrouter.ai/api/v1"

type StreamRequest struct {
	Model        string
	SystemPrompt string
	UserPrompt   string
	Tools        []ToolDefinition
}

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  any
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type ToolResult struct {
	ToolCallID string
	Name       string
	Content    string
}

type CompletionMetadata struct {
	FinishReason string
	TokensUsed   *int
}

type StreamHandlers struct {
	OnToken     func(text string)
	ExecuteTool func(ctx context.Context, call ToolCall) (ToolResult, error)
	OnComplete  func(metadata CompletionMetadata)
	OnError     func(code string, message string)
}

type StreamClient interface {
	Stream(ctx context.Context, request StreamRequest, handlers StreamHandlers) error
}

type Client struct {
	client *openai.Client
}

func NewClient(apiKey string) *Client {
	config := openai.DefaultConfig(strings.TrimSpace(apiKey))
	config.BaseURL = baseURL
	return &Client{client: openai.NewClientWithConfig(config)}
}

func (c *Client) Stream(ctx context.Context, request StreamRequest, handlers StreamHandlers) error {
	if strings.TrimSpace(request.Model) == "" {
		return errors.New("openrouter stream: model is required")
	}
	if strings.TrimSpace(request.UserPrompt) == "" {
		return errors.New("openrouter stream: user prompt is required")
	}

	messages := []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: request.SystemPrompt}, {Role: openai.ChatMessageRoleUser, Content: request.UserPrompt}}
	tools := toChatCompletionTools(request.Tools)

	for {
		stream, err := c.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
			Model:         request.Model,
			Messages:      messages,
			Tools:         tools,
			Stream:        true,
			StreamOptions: &openai.StreamOptions{IncludeUsage: true},
		})
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			if handlers.OnError != nil {
				handlers.OnError("provider_error", displaySafeProviderError(err))
			}
			return fmt.Errorf("create openrouter stream: %w", err)
		}

		finishReason := string(openai.FinishReasonStop)
		var tokensUsed *int
		var assistantContent strings.Builder
		toolCalls := make(map[int]ToolCall)
		toolCallOrder := make([]int, 0)

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				_ = stream.Close()
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return err
				}
				if handlers.OnError != nil {
					handlers.OnError("provider_error", displaySafeProviderError(err))
				}
				return fmt.Errorf("receive openrouter stream: %w", err)
			}
			if response.Usage != nil {
				totalTokens := response.Usage.TotalTokens
				tokensUsed = &totalTokens
			}

			var responseText strings.Builder
			for _, choice := range response.Choices {
				if choice.FinishReason != "" && choice.FinishReason != openai.FinishReasonNull {
					finishReason = string(choice.FinishReason)
				}
				if text := choice.Delta.Content; text != "" {
					assistantContent.WriteString(text)
					responseText.WriteString(text)
				}
				mergeToolCallChunks(toolCalls, &toolCallOrder, choice.Delta.ToolCalls)
			}
			if responseText.Len() > 0 && handlers.OnToken != nil {
				handlers.OnToken(responseText.String())
			}
		}
		if err := stream.Close(); err != nil {
			return err
		}

		orderedToolCalls := orderedToolCalls(toolCalls, toolCallOrder)
		assistantMessage := openai.ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant}
		if assistantContent.Len() > 0 {
			assistantMessage.Content = assistantContent.String()
		}
		if len(orderedToolCalls) > 0 {
			assistantMessage.ToolCalls = toAssistantToolCalls(orderedToolCalls)
		}
		if assistantMessage.Content != "" || len(assistantMessage.ToolCalls) > 0 {
			messages = append(messages, assistantMessage)
		}

		if finishReason != string(openai.FinishReasonToolCalls) {
			if handlers.OnComplete != nil {
				handlers.OnComplete(CompletionMetadata{
					FinishReason: finishReason,
					TokensUsed:   tokensUsed,
				})
			}
			return nil
		}
		if handlers.ExecuteTool == nil {
			return errors.New("openrouter stream: tool execution callback is required when the model requests tool calls")
		}

		for _, toolCall := range orderedToolCalls {
			result, err := handlers.ExecuteTool(ctx, toolCall)
			if err != nil {
				return fmt.Errorf("execute tool %s: %w", toolCall.Name, err)
			}
			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				ToolCallID: result.ToolCallID,
				Content:    result.Content,
			})
		}
	}
}

func toChatCompletionTools(definitions []ToolDefinition) []openai.Tool {
	if len(definitions) == 0 {
		return nil
	}

	tools := make([]openai.Tool, 0, len(definitions))
	for _, definition := range definitions {
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        definition.Name,
				Description: definition.Description,
				Parameters:  definition.Parameters,
			},
		})
	}
	return tools
}

func mergeToolCallChunks(accumulator map[int]ToolCall, order *[]int, chunks []openai.ToolCall) {
	for _, chunk := range chunks {
		index := len(accumulator)
		if chunk.Index != nil {
			index = *chunk.Index
		}

		call, exists := accumulator[index]
		if !exists {
			*order = append(*order, index)
		}
		if chunk.ID != "" {
			call.ID = chunk.ID
		}
		if chunk.Function.Name != "" {
			call.Name = chunk.Function.Name
		}
		if chunk.Function.Arguments != "" {
			call.Arguments += chunk.Function.Arguments
		}
		accumulator[index] = call
	}
}

func orderedToolCalls(accumulator map[int]ToolCall, order []int) []ToolCall {
	if len(order) == 0 {
		return nil
	}

	toolCalls := make([]ToolCall, 0, len(order))
	for _, index := range order {
		call, ok := accumulator[index]
		if !ok {
			continue
		}
		toolCalls = append(toolCalls, call)
	}
	return toolCalls
}

func toAssistantToolCalls(toolCalls []ToolCall) []openai.ToolCall {
	items := make([]openai.ToolCall, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		items = append(items, openai.ToolCall{
			ID:   toolCall.ID,
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionCall{
				Name:      toolCall.Name,
				Arguments: toolCall.Arguments,
			},
		})
	}
	return items
}

func displaySafeProviderError(err error) string {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "Relay could not complete the provider request."
	}
	message = strings.Join(strings.Fields(message), " ")
	lowerMessage := strings.ToLower(message)
	if isAuthenticationProviderError(lowerMessage) {
		return "Relay could not authenticate with OpenRouter. Check the saved API key and try again."
	}
	if containsSensitiveProviderError(lowerMessage) {
		return "Relay could not complete the provider request."
	}
	return "OpenRouter request failed: " + truncateProviderError(message)
}

func isAuthenticationProviderError(message string) bool {
	return strings.Contains(message, "api key") ||
		strings.Contains(message, "apikey") ||
		strings.Contains(message, "authentication") ||
		strings.Contains(message, "unauthorized") ||
		strings.Contains(message, "invalid key")
}

func containsSensitiveProviderError(message string) bool {
	return strings.Contains(message, "authorization") ||
		strings.Contains(message, "bearer ") ||
		strings.Contains(message, "x-api-key") ||
		strings.Contains(message, "secret") ||
		strings.Contains(message, "password") ||
		strings.Contains(message, "credential") ||
		strings.Contains(message, "sk-or-") ||
		strings.Contains(message, "sk-")
}

func truncateProviderError(message string) string {
	const maxLength = 240
	if len(message) <= maxLength {
		return message
	}
	return strings.TrimSpace(message[:maxLength-3]) + "..."
}
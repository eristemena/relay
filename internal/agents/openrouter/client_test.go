package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

func TestNewClientAndToolConversionHelpers(t *testing.T) {
	client := NewClient("  test-key  ")
	if client == nil || client.client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	definitions := []ToolDefinition{{
		Name:        "read_file",
		Description: "Read a file",
		Parameters: map[string]any{
			"path": "string",
		},
	}}
	tools := toChatCompletionTools(definitions)
	if len(tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(tools))
	}
	if tools[0].Function == nil || tools[0].Function.Name != "read_file" {
		t.Fatalf("tools[0].Function = %#v, want read_file definition", tools[0].Function)
	}

	assistantToolCalls := toAssistantToolCalls([]ToolCall{{
		ID:        "call_1",
		Name:      "read_file",
		Arguments: `{"path":"README.md"}`,
	}})
	if len(assistantToolCalls) != 1 {
		t.Fatalf("len(assistantToolCalls) = %d, want 1", len(assistantToolCalls))
	}
	if assistantToolCalls[0].Function.Name != "read_file" {
		t.Fatalf("assistantToolCalls[0].Function.Name = %q, want read_file", assistantToolCalls[0].Function.Name)
	}
}

func TestClientStreamRequiresModelAndUserPrompt(t *testing.T) {
	client := &Client{}

	if err := client.Stream(context.Background(), StreamRequest{UserPrompt: "hello"}, StreamHandlers{}); err == nil || err.Error() != "openrouter stream: model is required" {
		t.Fatalf("Stream() missing model error = %v, want model validation error", err)
	}

	err := client.Stream(context.Background(), StreamRequest{Model: "anthropic/claude-sonnet-4-5"}, StreamHandlers{})
	if err == nil || err.Error() != "openrouter stream: user prompt is required" {
		t.Fatalf("Stream() missing user prompt error = %v, want prompt validation error", err)
	}
}

func TestDisplaySafeProviderError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
	}{
		{
			name:    "empty error",
			err:     errors.New("   "),
			message: "Relay could not complete the provider request.",
		},
		{
			name:    "api key error",
			err:     errors.New("invalid api key"),
			message: "Relay could not authenticate with OpenRouter. Check the saved API key and try again.",
		},
		{
			name:    "generic provider error",
			err:     errors.New("upstream gateway timeout"),
			message: "OpenRouter request failed: upstream gateway timeout",
		},
		{
			name:    "sensitive provider error",
			err:     errors.New("provider rejected header authorization: Bearer sk-or-v1-secret"),
			message: "Relay could not complete the provider request.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := displaySafeProviderError(test.err); got != test.message {
				t.Fatalf("displaySafeProviderError() = %q, want %q", got, test.message)
			}
		})
	}
}

func TestMergeToolCallChunksReassemblesArgumentsByIndex(t *testing.T) {
	accumulator := make(map[int]ToolCall)
	order := make([]int, 0)
	indexZero := 0
	indexOne := 1

	mergeToolCallChunks(accumulator, &order, []openai.ToolCall{
		{
			Index: &indexZero,
			ID:    "call_0",
			Function: openai.FunctionCall{
				Name:      "read_file",
				Arguments: `{"path":"REA`,
			},
		},
		{
			Index: &indexOne,
			ID:    "call_1",
			Function: openai.FunctionCall{
				Name:      "search_codebase",
				Arguments: `{"query":"root"}`,
			},
		},
	})
	mergeToolCallChunks(accumulator, &order, []openai.ToolCall{
		{
			Index: &indexZero,
			Function: openai.FunctionCall{
				Arguments: `DME.md"}`,
			},
		},
	})

	toolCalls := orderedToolCalls(accumulator, order)
	if len(toolCalls) != 2 {
		t.Fatalf("len(toolCalls) = %d, want 2", len(toolCalls))
	}
	if toolCalls[0].Arguments != `{"path":"README.md"}` {
		t.Fatalf("toolCalls[0].Arguments = %q, want reconstructed README path", toolCalls[0].Arguments)
	}
	if toolCalls[1].Name != "search_codebase" {
		t.Fatalf("toolCalls[1].Name = %q, want search_codebase", toolCalls[1].Name)
	}
}

func TestClientStreamStreamsTokensToCompletion(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("request path = %q, want /chat/completions", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"id\":\"cmpl_1\",\"object\":\"chat.completion.chunk\",\"created\":0,\"model\":\"test-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"id\":\"cmpl_1\",\"object\":\"chat.completion.chunk\",\"created\":0,\"model\":\"test-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" world\"},\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-key")
	config.BaseURL = server.URL
	client := &Client{client: openai.NewClientWithConfig(config)}

	var tokens []string
	finishReason := ""
	err := client.Stream(context.Background(), StreamRequest{
		Model:      "test-model",
		UserPrompt: "Say hello",
	}, StreamHandlers{
		OnToken: func(text string) {
			tokens = append(tokens, text)
		},
		OnComplete: func(reason string) {
			finishReason = reason
		},
	})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if requestCount.Load() != 1 {
		t.Fatalf("requestCount = %d, want 1", requestCount.Load())
	}
	if got := strings.Join(tokens, ""); got != "Hello world" {
		t.Fatalf("joined tokens = %q, want Hello world", got)
	}
	if finishReason != "stop" {
		t.Fatalf("finishReason = %q, want stop", finishReason)
	}
}

func TestClientStreamExecutesToolCallsAndContinues(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callNumber := requestCount.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")

		var requestBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		switch callNumber {
		case 1:
			tools, ok := requestBody["tools"].([]any)
			if !ok || len(tools) != 1 {
				t.Fatalf("tools = %#v, want one tool definition", requestBody["tools"])
			}
			_, _ = w.Write([]byte("data: {\"id\":\"cmpl_2\",\"object\":\"chat.completion.chunk\",\"created\":0,\"model\":\"test-model\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\\\"REA\"}}]}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"id\":\"cmpl_2\",\"object\":\"chat.completion.chunk\",\"created\":0,\"model\":\"test-model\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"DME.md\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		case 2:
			messages, ok := requestBody["messages"].([]any)
			if !ok || len(messages) < 3 {
				t.Fatalf("messages = %#v, want assistant and tool follow-up messages", requestBody["messages"])
			}
			_, _ = w.Write([]byte("data: {\"id\":\"cmpl_3\",\"object\":\"chat.completion.chunk\",\"created\":0,\"model\":\"test-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Done\"},\"finish_reason\":\"stop\"}]}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		default:
			t.Fatalf("unexpected request count %d", callNumber)
		}
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-key")
	config.BaseURL = server.URL
	client := &Client{client: openai.NewClientWithConfig(config)}

	var executed []ToolCall
	finishReason := ""
	err := client.Stream(context.Background(), StreamRequest{
		Model:      "test-model",
		UserPrompt: "Read the README",
		Tools: []ToolDefinition{{
			Name:        "read_file",
			Description: "Read a file",
			Parameters: map[string]any{
				"path": "string",
			},
		}},
	}, StreamHandlers{
		ExecuteTool: func(_ context.Context, call ToolCall) (ToolResult, error) {
			executed = append(executed, call)
			return ToolResult{ToolCallID: call.ID, Name: call.Name, Content: "README content"}, nil
		},
		OnComplete: func(reason string) {
			finishReason = reason
		},
	})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if requestCount.Load() != 2 {
		t.Fatalf("requestCount = %d, want 2", requestCount.Load())
	}
	if !reflect.DeepEqual(executed, []ToolCall{{
		ID:        "call_1",
		Name:      "read_file",
		Arguments: `{"path":"README.md"}`,
	}}) {
		t.Fatalf("executed = %#v, want one merged tool call", executed)
	}
	if finishReason != "stop" {
		t.Fatalf("finishReason = %q, want stop", finishReason)
	}
}

func TestClientStreamBatchesTokenCallbacksPerChunk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"id\":\"cmpl_4\",\"object\":\"chat.completion.chunk\",\"created\":0,\"model\":\"test-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hel\"}},{\"index\":1,\"delta\":{\"content\":\"lo\"},\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-key")
	config.BaseURL = server.URL
	client := &Client{client: openai.NewClientWithConfig(config)}

	var tokens []string
	err := client.Stream(context.Background(), StreamRequest{
		Model:      "test-model",
		UserPrompt: "Say hello",
	}, StreamHandlers{
		OnToken: func(text string) {
			tokens = append(tokens, text)
		},
	})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if !reflect.DeepEqual(tokens, []string{"Hello"}) {
		t.Fatalf("tokens = %#v, want one batched callback", tokens)
	}
}

func TestClientStreamOmitsToolsForPromptOnlyRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		var requestBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if _, ok := requestBody["tools"]; ok {
			t.Fatalf("tools = %#v, want omitted tools for prompt-only request", requestBody["tools"])
		}

		_, _ = w.Write([]byte("data: {\"id\":\"cmpl_5\",\"object\":\"chat.completion.chunk\",\"created\":0,\"model\":\"test-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Done\"},\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-key")
	config.BaseURL = server.URL
	client := &Client{client: openai.NewClientWithConfig(config)}

	err := client.Stream(context.Background(), StreamRequest{
		Model:        "test-model",
		SystemPrompt: "You are prompt only.",
		UserPrompt:   "Summarize the orchestration state.",
	}, StreamHandlers{})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
}
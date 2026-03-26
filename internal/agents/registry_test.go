package agents

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/erisristemena/relay/internal/agents/openrouter"
	"github.com/erisristemena/relay/internal/config"
	"github.com/erisristemena/relay/internal/storage/sqlite"
)

func TestRegistrySelectProfileUsesRoleSpecificPromptsModelsAndToolAllowlists(t *testing.T) {
	registry := NewRegistry(config.AgentModels{})

	tests := []struct {
		name         string
		task         string
		wantRole     sqlite.AgentRole
		wantModel    string
		promptNeedle string
		policyNeedle string
		extraNeedle  string
		wantTools    []ToolName
	}{
		{
			name:         "planner",
			task:         "Plan the rollout steps for this migration",
			wantRole:     sqlite.RolePlanner,
			wantModel:    config.DefaultPlannerModel,
			promptNeedle: "Break developer tasks into a concrete sequence of steps",
			policyNeedle: "Do not ask the user for confirmation before taking allowed actions.",
			extraNeedle:  "Never ask questions such as 'Would you like me to search for related configuration files or documentation?'",
			wantTools:    []ToolName{ToolReadFile, ToolListFiles, ToolSearchCodebase, ToolGitLog, ToolGitDiff},
		},
		{
			name:         "coder",
			task:         "Implement the missing websocket state reducer",
			wantRole:     sqlite.RoleCoder,
			wantModel:    config.DefaultCoderModel,
			promptNeedle: "Produce focused implementation guidance",
			policyNeedle: "Do not ask the user for confirmation before taking allowed actions.",
			extraNeedle:  "Never ask questions such as 'Would you like me to proceed with writing these changes?'",
			wantTools:    []ToolName{ToolReadFile, ToolListFiles, ToolSearchCodebase, ToolGitLog, ToolGitDiff, ToolWriteFile, ToolRunCommand},
		},
		{
			name:         "reviewer",
			task:         "Review this patch for regressions",
			wantRole:     sqlite.RoleReviewer,
			wantModel:    config.DefaultReviewerModel,
			promptNeedle: "Prioritize correctness, regressions, missing tests, and security issues",
			policyNeedle: "Do not ask the user for confirmation before taking allowed actions.",
			extraNeedle:  "Never ask questions such as 'Would you like me to review this file?'",
			wantTools:    []ToolName{ToolReadFile, ToolListFiles, ToolSearchCodebase, ToolGitLog, ToolGitDiff},
		},
		{
			name:         "tester",
			task:         "Add test coverage for this handler",
			wantRole:     sqlite.RoleTester,
			wantModel:    config.DefaultTesterModel,
			promptNeedle: "Focus on validation strategy, failure modes",
			policyNeedle: "Do not ask the user for confirmation before taking allowed actions.",
			extraNeedle:  "The write_file tool is only for creating or updating recognized test files under tests/, __tests__/, or testdata/",
			wantTools:    []ToolName{ToolReadFile, ToolListFiles, ToolSearchCodebase, ToolGitLog, ToolGitDiff, ToolWriteFile, ToolRunCommand},
		},
		{
			name:         "explainer",
			task:         "Explain why the bootstrap logic restores this session",
			wantRole:     sqlite.RoleExplainer,
			wantModel:    config.DefaultExplainerModel,
			promptNeedle: "Explain code and architecture clearly",
			policyNeedle: "Do not ask the user for confirmation before taking allowed actions.",
			wantTools:    []ToolName{ToolReadFile, ToolListFiles, ToolSearchCodebase, ToolGitLog, ToolGitDiff},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := registry.SelectProfile(test.task)

			if profile.Role != test.wantRole {
				t.Fatalf("profile.Role = %q, want %q", profile.Role, test.wantRole)
			}
			if profile.Model != test.wantModel {
				t.Fatalf("profile.Model = %q, want %q", profile.Model, test.wantModel)
			}
			if !strings.Contains(profile.SystemPrompt, test.promptNeedle) {
				t.Fatalf("profile.SystemPrompt = %q, want substring %q", profile.SystemPrompt, test.promptNeedle)
			}
			if !strings.Contains(profile.SystemPrompt, test.policyNeedle) {
				t.Fatalf("profile.SystemPrompt = %q, want substring %q", profile.SystemPrompt, test.policyNeedle)
			}
			if test.extraNeedle != "" && !strings.Contains(profile.SystemPrompt, test.extraNeedle) {
				t.Fatalf("profile.SystemPrompt = %q, want substring %q", profile.SystemPrompt, test.extraNeedle)
			}
			if !reflect.DeepEqual(profile.AllowedTools, test.wantTools) {
				t.Fatalf("profile.AllowedTools = %v, want %v", profile.AllowedTools, test.wantTools)
			}
		})
	}
}

func TestRegistrySelectProfileUsesConfiguredModelOverrides(t *testing.T) {
	registry := NewRegistry(config.AgentModels{
		Planner:   "custom/planner",
		Coder:     "custom/coder",
		Reviewer:  "custom/reviewer",
		Tester:    "custom/tester",
		Explainer: "custom/explainer",
	})

	tests := []struct {
		name      string
		task      string
		wantModel string
	}{
		{name: "planner override", task: "design the migration plan", wantModel: "custom/planner"},
		{name: "coder override", task: "implement the handler", wantModel: "custom/coder"},
		{name: "reviewer override", task: "review the regression risk", wantModel: "custom/reviewer"},
		{name: "tester override", task: "test the websocket flow", wantModel: "custom/tester"},
		{name: "explainer override", task: "why does this reconnect work", wantModel: "custom/explainer"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := registry.SelectProfile(test.task)
			if profile.Model != test.wantModel {
				t.Fatalf("profile.Model = %q, want %q", profile.Model, test.wantModel)
			}
		})
	}
}

func TestRegistrySelectProfileFallsBackToCoderForGeneralTasks(t *testing.T) {
	profile := NewRegistry(config.AgentModels{}).SelectProfile("Tighten the websocket reconnect handling")
	if profile.Role != sqlite.RoleCoder {
		t.Fatalf("profile.Role = %q, want %q", profile.Role, sqlite.RoleCoder)
	}
}

func TestRegistryNewPromptOnlyAgentUsesRoleSpecificPromptOnlyProfiles(t *testing.T) {
	roles := []sqlite.AgentRole{
		sqlite.RolePlanner,
		sqlite.RoleCoder,
		sqlite.RoleReviewer,
		sqlite.RoleTester,
		sqlite.RoleExplainer,
	}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			fakeClient := &fakeStreamClient{}
			registry := NewRegistry(config.AgentModels{}).WithClientFactory(func(string) openrouter.StreamClient {
				return fakeClient
			})

			agent := registry.NewPromptOnlyAgent("or-test-key", role)
			profile := agent.Profile()
			if profile.Role != role {
				t.Fatalf("profile.Role = %q, want %q", profile.Role, role)
			}
			if len(profile.AllowedTools) != 0 {
				t.Fatalf("len(profile.AllowedTools) = %d, want 0 for prompt-only agent", len(profile.AllowedTools))
			}

			err := agent.Run(context.Background(), "Summarize this orchestration stage", StreamEventHandlers{})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if len(fakeClient.request.Tools) != 0 {
				t.Fatalf("len(request.Tools) = %d, want 0 for prompt-only agent", len(fakeClient.request.Tools))
			}
		})
	}
}

func TestRegistryNewAgentUsesRoleSpecificProfilesAndAdvertisesAllowedTools(t *testing.T) {
	roles := []sqlite.AgentRole{
		sqlite.RolePlanner,
		sqlite.RoleCoder,
		sqlite.RoleReviewer,
		sqlite.RoleTester,
		sqlite.RoleExplainer,
	}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			fakeClient := &fakeStreamClient{}
			registry := NewRegistry(config.AgentModels{}).WithClientFactory(func(string) openrouter.StreamClient {
				return fakeClient
			})
			executor := &fakeToolExecutor{}

			agent := registry.NewAgent("or-test-key", role, executor)
			profile := agent.Profile()
			if profile.Role != role {
				t.Fatalf("profile.Role = %q, want %q", profile.Role, role)
			}
			if len(profile.AllowedTools) == 0 {
				t.Fatal("profile.AllowedTools = empty, want role-specific tools")
			}

			err := agent.Run(context.Background(), "Inspect the repository before responding", StreamEventHandlers{})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if len(fakeClient.request.Tools) != len(profile.AllowedTools) {
				t.Fatalf("len(request.Tools) = %d, want %d", len(fakeClient.request.Tools), len(profile.AllowedTools))
			}
			for index, tool := range fakeClient.request.Tools {
				if tool.Name != string(profile.AllowedTools[index]) {
					t.Fatalf("request.Tools[%d].Name = %q, want %q", index, tool.Name, profile.AllowedTools[index])
				}
			}
		})
	}
}

func TestRegistryRunnerAdvertisesAndExecutesReadOnlyTools(t *testing.T) {
	fakeClient := &fakeStreamClient{}
	registry := NewRegistry(config.AgentModels{}).WithClientFactory(func(string) openrouter.StreamClient {
		return fakeClient
	})
	executor := &fakeToolExecutor{}
	runner := registry.NewRunner("or-test-key", "Explain the websocket reconnect flow", executor)

	toolCalls := make([]ToolCallEvent, 0, 1)
	toolResults := make([]ToolResultEvent, 0, 1)
	err := runner.Run(context.Background(), "Explain the websocket reconnect flow", StreamEventHandlers{
		OnToolCall: func(event ToolCallEvent) {
			toolCalls = append(toolCalls, event)
		},
		OnToolResult: func(event ToolResultEvent) {
			toolResults = append(toolResults, event)
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(fakeClient.request.Tools) != 5 {
		t.Fatalf("len(request.Tools) = %d, want 5", len(fakeClient.request.Tools))
	}
	if fakeClient.request.Tools[0].Name != string(ToolReadFile) {
		t.Fatalf("request.Tools[0].Name = %q, want %q", fakeClient.request.Tools[0].Name, ToolReadFile)
	}
	if fakeClient.request.Tools[1].Name != string(ToolListFiles) {
		t.Fatalf("request.Tools[1].Name = %q, want %q", fakeClient.request.Tools[1].Name, ToolListFiles)
	}
	if fakeClient.request.Tools[2].Name != string(ToolSearchCodebase) {
		t.Fatalf("request.Tools[2].Name = %q, want %q", fakeClient.request.Tools[2].Name, ToolSearchCodebase)
	}
	if fakeClient.request.Tools[3].Name != string(ToolGitLog) {
		t.Fatalf("request.Tools[3].Name = %q, want %q", fakeClient.request.Tools[3].Name, ToolGitLog)
	}
	if fakeClient.request.Tools[4].Name != string(ToolGitDiff) {
		t.Fatalf("request.Tools[4].Name = %q, want %q", fakeClient.request.Tools[4].Name, ToolGitDiff)
	}
	if len(toolCalls) != 1 {
		t.Fatalf("len(toolCalls) = %d, want 1", len(toolCalls))
	}
	if len(toolResults) != 1 {
		t.Fatalf("len(toolResults) = %d, want 1", len(toolResults))
	}
	if toolCalls[0].ToolName != ToolReadFile {
		t.Fatalf("toolCalls[0].ToolName = %q, want %q", toolCalls[0].ToolName, ToolReadFile)
	}
	if toolResults[0].Status != "completed" {
		t.Fatalf("toolResults[0].Status = %q, want completed", toolResults[0].Status)
	}
	if executor.executedName != ToolReadFile {
		t.Fatalf("executedName = %q, want %q", executor.executedName, ToolReadFile)
	}
	if !strings.Contains(executor.executedArgs, `"path":"README.md"`) {
		t.Fatalf("executedArgs = %q, want serialized path", executor.executedArgs)
	}
	if fakeClient.toolResult.Content != "file contents" {
		t.Fatalf("toolResult.Content = %q, want file contents", fakeClient.toolResult.Content)
	}
	if got := toolCalls[0].InputPreview["path"]; got != "README.md" {
		t.Fatalf("toolCalls[0].InputPreview[path] = %v, want README.md", got)
	}
	if got := toolResults[0].ResultPreview["summary"]; got != "Loaded file content." {
		t.Fatalf("toolResults[0].ResultPreview[summary] = %v, want Loaded file content.", got)
	}
}

type fakeStreamClient struct {
	request    openrouter.StreamRequest
	toolResult openrouter.ToolResult
}

func (f *fakeStreamClient) Stream(ctx context.Context, request openrouter.StreamRequest, handlers openrouter.StreamHandlers) error {
	f.request = request
	if handlers.ExecuteTool == nil {
		return nil
	}

	result, err := handlers.ExecuteTool(ctx, openrouter.ToolCall{
		ID:        "call_123",
		Name:      string(ToolReadFile),
		Arguments: `{"path":"README.md"}`,
	})
	if err != nil {
		return err
	}
	f.toolResult = result
	if handlers.OnComplete != nil {
		handlers.OnComplete("stop")
	}
	return nil
}

type fakeToolExecutor struct {
	executedName ToolName
	executedArgs string
}

func (f *fakeToolExecutor) Definitions(allowedTools []ToolName) []ToolDefinition {
	definitions := make([]ToolDefinition, 0, len(allowedTools))
	for _, tool := range allowedTools {
		definitions = append(definitions, ToolDefinition{
			Name:        tool,
			Description: "Tool definition for tests.",
			Parameters:  map[string]any{"type": "object"},
		})
	}
	return definitions
}

func (f *fakeToolExecutor) PreviewToolCall(_ ToolName, arguments json.RawMessage) map[string]any {
	return map[string]any{"path": "README.md"}
}

func (f *fakeToolExecutor) ExecuteTool(_ context.Context, toolCallID string, name ToolName, arguments json.RawMessage) (ToolExecutionResult, error) {
	f.executedName = name
	f.executedArgs = string(arguments)
	return ToolExecutionResult{
		ToolCallID: toolCallID,
		ToolName:   name,
		Status:     "completed",
		Content:    "file contents",
		ResultPreview: map[string]any{
			"summary": "Loaded file content.",
		},
	}, nil
}

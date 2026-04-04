package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/eristemena/relay/internal/storage/sqlite"
)

type RunHistoryExportRequest struct {
	SessionID   string
	RunID       string
	DirectUser  bool
	RequestedAt time.Time
}

type RunHistoryExportResult struct {
	SessionID   string    `json:"session_id"`
	RunID       string    `json:"run_id"`
	Status      string    `json:"status"`
	ExportPath  string    `json:"export_path,omitempty"`
	GeneratedAt time.Time `json:"generated_at,omitempty"`
}

var exportSlugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

func (s *Service) ExportRunHistory(ctx context.Context, input RunHistoryExportRequest) (RunHistoryExportResult, error) {
	if !input.DirectUser {
		return RunHistoryExportResult{}, fmt.Errorf("run history export requires a direct user request")
	}
	if err := s.refreshRunHistory(ctx, input.RunID); err != nil {
		return RunHistoryExportResult{}, err
	}
	run, err := s.store.GetAgentRun(ctx, input.RunID)
	if err != nil {
		return RunHistoryExportResult{}, err
	}
	if run.SessionID != input.SessionID {
		return RunHistoryExportResult{}, fmt.Errorf("run %s does not belong to session %s", input.RunID, input.SessionID)
	}
	document, err := s.store.GetRunHistoryDocument(ctx, input.RunID)
	if err != nil {
		return RunHistoryExportResult{}, fmt.Errorf("load run history document: %w", err)
	}
	changeRecords, err := s.store.ListRunChangeRecords(ctx, input.RunID)
	if err != nil {
		return RunHistoryExportResult{}, fmt.Errorf("load run history changes: %w", err)
	}
	events, err := s.store.ListRunEvents(ctx, input.RunID)
	if err != nil {
		return RunHistoryExportResult{}, fmt.Errorf("load run history events: %w", err)
	}
	exportDir := filepath.Join(s.paths.ConfigDir, "exports")
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		return RunHistoryExportResult{}, fmt.Errorf("create exports directory: %w", err)
	}
	generatedAt := input.RequestedAt.UTC()
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	exportPath, err := buildRunHistoryExportPath(exportDir, generatedAt, document.GeneratedTitle, input.RunID)
	if err != nil {
		return RunHistoryExportResult{}, fmt.Errorf("build export path: %w", err)
	}
	participants, _ := deriveParticipants(run, mustListAgentExecutions(ctx, s, input.RunID), events)
	markdown := buildRunHistoryExportMarkdown(document, changeRecords, events)
	if err := os.WriteFile(exportPath, []byte(markdown), 0o644); err != nil {
		return RunHistoryExportResult{}, fmt.Errorf("write export markdown: %w", err)
	}
	exportDocument := sqlite.RunExportDocument{
		RunID:            input.RunID,
		ExportPath:       exportPath,
		GeneratedAt:      generatedAt,
		Title:            document.GeneratedTitle,
		FinalStatus:      document.FinalStatus,
		Participants:     splitParticipants(participants),
		TimelineMarkdown: buildTimelineMarkdown(events),
		ChangesMarkdown:  buildChangesMarkdown(changeRecords),
		RequestedBy:      "developer",
	}
	if err := s.store.RecordRunExport(ctx, exportDocument); err != nil {
		return RunHistoryExportResult{}, fmt.Errorf("record run export: %w", err)
	}
	return RunHistoryExportResult{
		SessionID:   input.SessionID,
		RunID:       input.RunID,
		Status:      "completed",
		ExportPath:  exportPath,
		GeneratedAt: generatedAt,
	}, nil
}

func mustListAgentExecutions(ctx context.Context, s *Service, runID string) []sqlite.AgentExecution {
	executions, err := s.store.ListAgentExecutions(ctx, runID)
	if err != nil {
		return nil
	}
	return executions
}

func buildRunHistoryExportFilename(generatedAt time.Time, title string, runID string) string {
	slug := strings.ToLower(strings.TrimSpace(title))
	if slug == "" {
		slug = runID
	}
	slug = exportSlugSanitizer.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = runID
	}
	return fmt.Sprintf("%s-%s-%s.md", generatedAt.Format("2006-01-02-150405-000000000"), runID, slug)
	}

func buildRunHistoryExportPath(exportDir string, generatedAt time.Time, title string, runID string) (string, error) {
	baseFilename := buildRunHistoryExportFilename(generatedAt, title, runID)
	basePath := filepath.Join(exportDir, baseFilename)
	if _, err := os.Stat(basePath); err != nil {
		if os.IsNotExist(err) {
			return basePath, nil
		}
		return "", err
	}

	ext := filepath.Ext(baseFilename)
	name := strings.TrimSuffix(baseFilename, ext)
	for suffix := 2; suffix < 10_000; suffix++ {
		candidatePath := filepath.Join(exportDir, fmt.Sprintf("%s-%d%s", name, suffix, ext))
		if _, err := os.Stat(candidatePath); err != nil {
			if os.IsNotExist(err) {
				return candidatePath, nil
			}
			return "", err
		}
	}

	return "", fmt.Errorf("unable to allocate unique export filename for run %s", runID)
}

func buildRunHistoryExportMarkdown(document sqlite.RunHistoryDocument, changeRecords []sqlite.RunChangeRecord, events []sqlite.AgentRunEvent) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(document.GeneratedTitle)
	builder.WriteString("\n\n")
	builder.WriteString("- Run ID: `")
	builder.WriteString(document.RunID)
	builder.WriteString("`\n")
	builder.WriteString("- Final status: ")
	builder.WriteString(document.FinalStatus)
	builder.WriteString("\n")
	builder.WriteString("- Started at: ")
	builder.WriteString(document.StartedAt.UTC().Format(time.RFC3339))
	builder.WriteString("\n")
	if document.CompletedAt != nil {
		builder.WriteString("- Completed at: ")
		builder.WriteString(document.CompletedAt.UTC().Format(time.RFC3339))
		builder.WriteString("\n")
	}
	builder.WriteString("- Agent count: ")
	builder.WriteString(fmt.Sprintf("%d", document.AgentCount))
	builder.WriteString("\n\n")
	builder.WriteString("## Goal\n\n")
	builder.WriteString(document.GoalText)
	builder.WriteString("\n\n")
	if strings.TrimSpace(document.SummaryText) != "" {
		builder.WriteString("## Summary\n\n")
		builder.WriteString(document.SummaryText)
		builder.WriteString("\n\n")
	}
	builder.WriteString("## Timeline\n\n")
	builder.WriteString(buildTimelineMarkdown(events))
	if changes := buildChangesMarkdown(changeRecords); strings.TrimSpace(changes) != "" {
		builder.WriteString("\n\n## Changes\n\n")
		builder.WriteString(changes)
	}
	return builder.String()
}

func buildTimelineMarkdown(events []sqlite.AgentRunEvent) string {
	if len(events) == 0 {
		return "No preserved timeline events were recorded."
	}
	lines := make([]string, 0, len(events))
	for _, event := range events {
		occurredAt, ok, _ := replayOccurredAt(event)
		if !ok {
			occurredAt = event.CreatedAt
		}
		payload := decodeEventPayload(event)
		message := summarizeReplayEvent(event.EventType, payload)
		lines = append(lines, fmt.Sprintf("- %s `%s`: %s", occurredAt.UTC().Format(time.RFC3339), event.EventType, message))
	}
	return strings.Join(lines, "\n")
}

func buildChangesMarkdown(changeRecords []sqlite.RunChangeRecord) string {
	if len(changeRecords) == 0 {
		return ""
	}
	sections := make([]string, 0, len(changeRecords))
	for _, record := range changeRecords {
		section := []string{
			fmt.Sprintf("### %s", record.Path),
			fmt.Sprintf("- Tool call: `%s`", record.ToolCallID),
			fmt.Sprintf("- Approval state: %s", record.ApprovalState),
			"",
			"#### Before",
			"```text",
			record.OriginalContent,
			"```",
			"",
			"#### After",
			"```text",
			record.ProposedContent,
			"```",
		}
		sections = append(sections, strings.Join(section, "\n"))
	}
	return strings.Join(sections, "\n\n")
}

func summarizeReplayEvent(eventType string, payload map[string]any) string {
	for _, key := range []string{"message", "summary", "text", "state", "finish_reason", "tool_name", "task_text", "code"} {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "Recorded event"
}

func splitParticipants(participantText string) []string {
	trimmed := strings.TrimSpace(participantText)
	if trimmed == "" {
		return nil
	}
	return strings.Fields(trimmed)
}
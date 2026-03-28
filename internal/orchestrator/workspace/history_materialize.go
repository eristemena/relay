package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/erisristemena/relay/internal/storage/sqlite"
)

func (s *Service) refreshSessionRunHistory(ctx context.Context, sessionID string) error {
	runs, err := s.store.ListRunSummaries(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("list session runs for history refresh: %w", err)
	}
	for _, run := range runs {
		if err := s.refreshRunHistory(ctx, run.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) refreshRunHistory(ctx context.Context, runID string) error {
	run, err := s.store.GetAgentRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("load run for history refresh: %w", err)
	}
	events, err := s.store.ListRunEvents(ctx, runID)
	if err != nil {
		return fmt.Errorf("list run events for history refresh: %w", err)
	}
	executions, err := s.store.ListAgentExecutions(ctx, runID)
	if err != nil {
		return fmt.Errorf("list executions for history refresh: %w", err)
	}
	approvals, err := s.store.ListApprovalRequestsForRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("list approvals for history refresh: %w", err)
	}

	changeRecords := buildRunChangeRecords(approvals)
	if err := s.store.ReplaceRunChangeRecords(ctx, runID, changeRecords); err != nil {
		return fmt.Errorf("replace run change records: %w", err)
	}

	generatedTitle := deriveGeneratedTitle(run.TaskText, events)
	summaryText := deriveSummaryText(events)
	firstEventAt, lastEventAt := deriveEventBounds(events)
	participantText, agentCount := deriveParticipants(run, executions, events)
	hasFileChanges := len(changeRecords) > 0
	document := sqlite.RunHistoryDocument{
		RunID:            run.ID,
		SessionID:        run.SessionID,
		GeneratedTitle:   generatedTitle,
		GoalText:         run.TaskText,
		FinalStatus:      deriveFinalStatus(run),
		AgentCount:       agentCount,
		StartedAt:        run.StartedAt,
		CompletedAt:      run.CompletedAt,
		FirstEventAt:     firstEventAt,
		LastEventAt:      lastEventAt,
		SummaryText:      summaryText,
		TouchedFileCount: len(changeRecords),
		HasFileChanges:   hasFileChanges,
	}
	if existingExport, err := s.store.GetLatestRunExport(ctx, runID); err == nil {
		document.ExportedAt = &existingExport.GeneratedAt
	}
	if err := s.store.UpsertRunHistoryDocument(ctx, document); err != nil {
		return fmt.Errorf("upsert run history document: %w", err)
	}

	searchDocument := sqlite.RunHistorySearchDocument{
		RunID:           run.ID,
		SessionID:       run.SessionID,
		TitleText:       generatedTitle,
		GoalText:        run.TaskText,
		SummaryText:     summaryText,
		TranscriptText:  deriveTranscriptText(events),
		FileNamesText:   deriveFileNamesText(changeRecords),
		ParticipantText: participantText,
	}
	if err := s.store.UpsertRunHistorySearchDocument(ctx, searchDocument); err != nil {
		return fmt.Errorf("upsert run history search document: %w", err)
	}

	return nil
}

func buildRunChangeRecords(approvals []sqlite.ApprovalRequest) []sqlite.RunChangeRecord {
	records := make([]sqlite.RunChangeRecord, 0)
	for _, approval := range approvals {
		summary, err := approvalSummaryFromRecord(approval)
		if err != nil {
			continue
		}
		if summary.DiffPreview == nil {
			continue
		}
		path, _ := summary.DiffPreview["target_path"].(string)
		baseHash, _ := summary.DiffPreview["base_content_hash"].(string)
		originalContent, _ := summary.DiffPreview["original_content"].(string)
		proposedContent, _ := summary.DiffPreview["proposed_content"].(string)
		if strings.TrimSpace(path) == "" || strings.TrimSpace(baseHash) == "" {
			continue
		}
		records = append(records, sqlite.RunChangeRecord{
			RunID:           approval.RunID,
			ToolCallID:      approval.ToolCallID,
			Path:            path,
			OriginalContent: originalContent,
			ProposedContent: proposedContent,
			BaseContentHash: baseHash,
			ApprovalState:   approval.State,
			Role:            approval.Role,
			Model:           approval.Model,
			OccurredAt:      approval.OccurredAt,
		})
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].OccurredAt.Equal(records[j].OccurredAt) {
			if records[i].ToolCallID == records[j].ToolCallID {
				return records[i].Path < records[j].Path
			}
			return records[i].ToolCallID < records[j].ToolCallID
		}
		return records[i].OccurredAt.Before(records[j].OccurredAt)
	})
	return records
}

func deriveGeneratedTitle(taskText string, events []sqlite.AgentRunEvent) string {
	if summary := strings.TrimSpace(deriveSummaryText(events)); summary != "" {
		return truncateSingleLine(summary, 72)
	}
	return truncateSingleLine(taskText, 72)
}

func deriveSummaryText(events []sqlite.AgentRunEvent) string {
	for index := len(events) - 1; index >= 0; index-- {
		payload := decodeEventPayload(events[index])
		if summary, ok := payload["summary"].(string); ok && strings.TrimSpace(summary) != "" {
			return strings.TrimSpace(summary)
		}
	}
	return ""
}

func deriveTranscriptText(events []sqlite.AgentRunEvent) string {
	var builder strings.Builder
	for _, event := range events {
		if event.EventType != sqlite.EventTypeToken {
			continue
		}
		payload := decodeEventPayload(event)
		text, _ := payload["text"].(string)
		if strings.TrimSpace(text) == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(strings.TrimSpace(text))
	}
	return builder.String()
}

func deriveEventBounds(events []sqlite.AgentRunEvent) (*time.Time, *time.Time) {
	var first *time.Time
	var last *time.Time
	for _, event := range events {
		occurredAt, ok, _ := replayOccurredAt(event)
		if !ok {
			continue
		}
		if first == nil || occurredAt.Before(*first) {
			copy := occurredAt
			first = &copy
		}
		if last == nil || occurredAt.After(*last) {
			copy := occurredAt
			last = &copy
		}
	}
	return first, last
}

func deriveParticipants(run sqlite.AgentRun, executions []sqlite.AgentExecution, events []sqlite.AgentRunEvent) (string, int) {
	participants := map[string]struct{}{}
	recordedAgents := map[string]struct{}{}
	addParticipant := func(role sqlite.AgentRole, model string) {
		parts := []string{}
		if strings.TrimSpace(string(role)) != "" {
			parts = append(parts, strings.TrimSpace(string(role)))
		}
		if strings.TrimSpace(model) != "" {
			parts = append(parts, strings.TrimSpace(model))
		}
		if len(parts) == 0 {
			return
		}
		participants[strings.Join(parts, " ")] = struct{}{}
	}
	addRecordedAgent := func(agentID string) {
		trimmed := strings.TrimSpace(agentID)
		if trimmed == "" {
			return
		}
		recordedAgents[trimmed] = struct{}{}
	}

	addParticipant(run.Role, run.Model)
	for _, execution := range executions {
		addParticipant(execution.Role, execution.Model)
		addRecordedAgent(execution.ID)
	}
	for _, event := range events {
		addParticipant(event.Role, event.Model)
		payload := decodeEventPayload(event)
		agentID, _ := payload["agent_id"].(string)
		addRecordedAgent(agentID)
	}

	keys := make([]string, 0, len(participants))
	for key := range participants {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	agentCount := len(recordedAgents)
	if agentCount == 0 {
		agentCount = 1
	}

	return strings.Join(keys, " "), agentCount
}

func deriveFileNamesText(records []sqlite.RunChangeRecord) string {
	parts := make([]string, 0, len(records)*2)
	for _, record := range records {
		trimmed := strings.TrimSpace(record.Path)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
		segments := strings.Split(trimmed, "/")
		parts = append(parts, segments[len(segments)-1])
	}
	return strings.Join(parts, " ")
}

func deriveFinalStatus(run sqlite.AgentRun) string {
	if strings.HasSuffix(strings.TrimSpace(run.ErrorCode), "_clarification_required") {
		return "clarification_required"
	}
	switch run.State {
	case sqlite.RunStateCompleted:
		return "completed"
	case sqlite.RunStateHalted:
		return "halted"
	case sqlite.RunStateErrored, sqlite.RunStateCancelled:
		return "errored"
	default:
		return strings.TrimSpace(run.State)
	}
}

func truncateSingleLine(value string, limit int) string {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if trimmed == "" {
		return "Untitled run"
	}
	if len(trimmed) <= limit {
		return trimmed
	}
	if limit <= 3 {
		return trimmed[:limit]
	}
	return trimmed[:limit-3] + "..."
}

func decodeEventPayload(event sqlite.AgentRunEvent) map[string]any {
	var payload map[string]any
	if err := json.Unmarshal([]byte(event.PayloadJSON), &payload); err != nil {
		return map[string]any{}
	}
	return payload
}
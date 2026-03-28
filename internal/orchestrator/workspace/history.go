package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erisristemena/relay/internal/storage/sqlite"
)

const replaySyntheticStepMS = 25

func (s *Service) OpenRun(ctx context.Context, input OpenRunInput, emit func(StreamEnvelope) error) (WorkspaceSnapshot, error) {
	run, err := s.store.GetAgentRun(ctx, input.RunID)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	if run.SessionID != input.SessionID {
		return WorkspaceSnapshot{}, fmt.Errorf("run %s does not belong to session %s", input.RunID, input.SessionID)
	}

	subscriberID := ""
	if run.Active() {
		if id, attached := s.attachRunSubscriber(ctx, run.ID, emit, true); attached {
			subscriberID = id
		}
	}

	snapshot, err := s.Bootstrap(ctx, input.SessionID)
	if err != nil {
		if subscriberID != "" {
			s.detachRunSubscriber(run.ID, subscriberID)
		}
		return WorkspaceSnapshot{}, err
	}

	events, err := s.store.ListRunEvents(ctx, input.RunID)
	if err != nil {
		if subscriberID != "" {
			s.detachRunSubscriber(run.ID, subscriberID)
		}
		return WorkspaceSnapshot{}, err
	}
	var maxSequence int64
	timeline := buildReplayTimeline(run.ID, events)
	firstOccurredAt, lastOccurredAt, hasOccurredAt := replayTimelineBounds(timeline)
	durationMS := timeline.DurationMS
	approvals, err := s.store.ListPendingApprovalRequestsForRun(ctx, run.ID)
	if err != nil {
		if subscriberID != "" {
			s.detachRunSubscriber(run.ID, subscriberID)
		}
		return WorkspaceSnapshot{}, fmt.Errorf("load pending approvals: %w", err)
	}
	finalStatus := "completed"
	if run.Active() {
		finalStatus = "paused"
	}
	s.storeReplaySession(input.SessionID, run.ID, timeline, approvals, ReplaySession{
		SessionID:   replaySessionKey(input.SessionID, run.ID),
		WorkspaceID: input.SessionID,
		RunID:       run.ID,
		Status:      finalStatus,
		Speed:       1,
		CursorMS:    durationMS,
		DurationMS:  durationMS,
	})
	if err := emit(StreamEnvelope{Type: "agent.run.replay.state", Payload: replayStatePayload(input.SessionID, run.ID, "preparing", 0, durationMS, 1, firstOccurredAt, hasOccurredAt)}); err != nil {
		if subscriberID != "" {
			s.detachRunSubscriber(run.ID, subscriberID)
		}
		return WorkspaceSnapshot{}, err
	}
	for _, event := range timeline.Events {
		if event.Sequence > maxSequence {
			maxSequence = event.Sequence
		}
		if err := emit(StreamEnvelope{Type: event.Type, Payload: cloneReplayPayload(event)}); err != nil {
			if subscriberID != "" {
				s.detachRunSubscriber(run.ID, subscriberID)
			}
			return WorkspaceSnapshot{}, err
		}
	}

	if subscriberID != "" {
		if err := s.finishReplay(run.ID, subscriberID, maxSequence); err != nil {
			return WorkspaceSnapshot{}, err
		}
	}

	payloads := make([]map[string]any, 0, len(approvals))
	for _, approval := range approvals {
		summary, err := approvalSummaryFromRecord(approval)
		if err != nil {
			if subscriberID != "" {
				s.detachRunSubscriber(run.ID, subscriberID)
			}
			return WorkspaceSnapshot{}, err
		}
		payloads = append(payloads, approvalPayload(summary))
	}
	for _, payload := range payloads {
		if err := emit(StreamEnvelope{Type: "approval_request", Payload: payload}); err != nil {
			if subscriberID != "" {
				s.detachRunSubscriber(run.ID, subscriberID)
			}
			return WorkspaceSnapshot{}, err
		}
	}

	finalCursorMS := durationMS
	finalTimestamp := lastOccurredAt
	if !hasOccurredAt {
		finalTimestamp = time.Time{}
	}
	if err := emit(StreamEnvelope{Type: "agent.run.replay.state", Payload: replayStatePayload(input.SessionID, run.ID, finalStatus, finalCursorMS, durationMS, 1, finalTimestamp, hasOccurredAt)}); err != nil {
		if subscriberID != "" {
			s.detachRunSubscriber(run.ID, subscriberID)
		}
		return WorkspaceSnapshot{}, err
	}

	return snapshot, nil
}

func buildReplayTimeline(runID string, events []sqlite.AgentRunEvent) replayTimeline {
	timeline := replayTimeline{RunID: runID, Events: make([]replayEvent, 0, len(events))}
	var first time.Time
	hasFirst := false
	lastRelativeMS := 0
	for _, event := range events {
		payload := decodeEventPayload(event)
		occurredAt, ok, hasSubsecondPrecision := replayOccurredAt(event)
		if !ok {
			occurredAt = event.CreatedAt
		}
		if !hasFirst {
			first = occurredAt
			hasFirst = true
		}
		relativeMS := 0
		if hasFirst {
			relativeMS = int(occurredAt.Sub(first).Milliseconds())
			if relativeMS < 0 {
				relativeMS = 0
			}
		}
		if len(timeline.Events) > 0 && relativeMS <= lastRelativeMS {
			if relativeMS < lastRelativeMS || !hasSubsecondPrecision {
				relativeMS = lastRelativeMS + replaySyntheticStepMS
			}
		}
		payload["sequence"] = event.Sequence
		payload["replay"] = true
		if event.TokensUsed != nil {
			payload["tokens_used"] = *event.TokensUsed
		}
		if event.ContextLimit != nil {
			payload["context_limit"] = *event.ContextLimit
		}
		timeline.Events = append(timeline.Events, replayEvent{Sequence: event.Sequence, OccurredAt: occurredAt, Type: event.EventType, Payload: payload, RelativeMS: relativeMS})
		lastRelativeMS = relativeMS
		timeline.DurationMS = relativeMS
	}
	return timeline
}

func replayTimelineBounds(timeline replayTimeline) (time.Time, time.Time, bool) {
	if len(timeline.Events) == 0 {
		return time.Time{}, time.Time{}, false
	}
	return timeline.Events[0].OccurredAt, timeline.Events[len(timeline.Events)-1].OccurredAt, true
}

func cloneReplayPayload(event replayEvent) map[string]any {
	cloned := make(map[string]any, len(event.Payload))
	for key, value := range event.Payload {
		cloned[key] = value
	}
	return cloned
}

func replayOccurredAt(event sqlite.AgentRunEvent) (time.Time, bool, bool) {
	if !event.CreatedAt.IsZero() {
		var payload map[string]any
		if err := json.Unmarshal([]byte(event.PayloadJSON), &payload); err == nil {
			if rawOccurredAt, ok := payload["occurred_at"].(string); ok {
				if occurredAt, err := parseEventTimestamp(rawOccurredAt); err == nil {
					return occurredAt, true, timestampHasSubsecondPrecision(rawOccurredAt)
				}
			}
		}
		return event.CreatedAt, true, event.CreatedAt.Nanosecond() != 0
	}
	return time.Time{}, false, false
}

func replayStatePayload(sessionID string, runID string, status string, cursorMS int, durationMS int, speed float64, selectedTimestamp time.Time, hasTimestamp bool) map[string]any {
	payload := map[string]any{
		"session_id":  sessionID,
		"run_id":      runID,
		"status":      status,
		"cursor_ms":   cursorMS,
		"duration_ms": durationMS,
		"speed":       speed,
	}
	if hasTimestamp {
		payload["selected_timestamp"] = formatEventTimestamp(selectedTimestamp)
	}
	return payload
}

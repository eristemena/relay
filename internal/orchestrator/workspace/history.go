package workspace

import (
	"context"
	"encoding/json"
	"fmt"
)

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
	for _, event := range events {
		var payload map[string]any
		if err := json.Unmarshal([]byte(event.PayloadJSON), &payload); err != nil {
			if subscriberID != "" {
				s.detachRunSubscriber(run.ID, subscriberID)
			}
			return WorkspaceSnapshot{}, fmt.Errorf("decode stored run event: %w", err)
		}
		payload["sequence"] = event.Sequence
		payload["replay"] = true
		if event.Sequence > maxSequence {
			maxSequence = event.Sequence
		}
		if err := emit(StreamEnvelope{Type: event.EventType, Payload: payload}); err != nil {
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
		payloads, err := s.pendingApprovalPayloads(ctx, run.ID)
		if err != nil {
			s.detachRunSubscriber(run.ID, subscriberID)
			return WorkspaceSnapshot{}, fmt.Errorf("load pending approvals: %w", err)
		}
		for _, payload := range payloads {
			if err := emit(StreamEnvelope{Type: "approval_request", Payload: payload}); err != nil {
				s.detachRunSubscriber(run.ID, subscriberID)
				return WorkspaceSnapshot{}, err
			}
		}
	}

	return snapshot, nil
}
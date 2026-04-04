package workspace

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eristemena/relay/internal/agents"
	"github.com/eristemena/relay/internal/config"
	"github.com/eristemena/relay/internal/storage/sqlite"
)

func TestService_OpenRunReplaysHistoricalApprovalRequests(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Replay approval-required run")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Wait for approval", sqlite.RoleCoder, config.DefaultCoderModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateHalted
	run.ErrorCode = "approval_required"
	run.ErrorMessage = "Relay paused until the developer reviews the proposed change."
	run.CompletedAt = &completedAt
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeStateChange, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","state":"tool_running","message":"Running write_file","occurred_at":"2026-03-24T12:00:00Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() state_change error = %v", err)
	}
	inputPreviewJSON, err := json.Marshal(map[string]any{
		"path": "README.md",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if _, err := store.CreateApprovalRequest(ctx, sqlite.ApprovalRequest{
		SessionID:        session.ID,
		RunID:            run.ID,
		ToolCallID:       "call_approval_1",
		ToolName:         string(agents.ToolWriteFile),
		Role:             sqlite.RoleCoder,
		Model:            config.DefaultCoderModel,
		InputPreviewJSON: string(inputPreviewJSON),
		Message:          "Relay needs approval before it can write files inside the configured project root.",
		State:            sqlite.ApprovalStateProposed,
		OccurredAt:       completedAt,
	}); err != nil {
		t.Fatalf("CreateApprovalRequest() error = %v", err)
	}

	envelopes := make([]StreamEnvelope, 0, 2)
	_, err = service.OpenRun(ctx, OpenRunInput{SessionID: session.ID, RunID: run.ID}, func(envelope StreamEnvelope) error {
		envelopes = append(envelopes, envelope)
		return nil
	})
	if err != nil {
		t.Fatalf("OpenRun() error = %v", err)
	}
	if len(envelopes) != 4 {
		t.Fatalf("len(envelopes) = %d, want 4", len(envelopes))
	}
	if envelopes[0].Type != "agent.run.replay.state" {
		t.Fatalf("envelopes[0].Type = %q, want agent.run.replay.state", envelopes[0].Type)
	}
	if envelopes[1].Type != sqlite.EventTypeStateChange {
		t.Fatalf("envelopes[1].Type = %q, want %q", envelopes[1].Type, sqlite.EventTypeStateChange)
	}
	if envelopes[2].Type != "approval_request" {
		t.Fatalf("envelopes[2].Type = %q, want approval_request", envelopes[2].Type)
	}
	if envelopes[3].Type != "agent.run.replay.state" {
		t.Fatalf("envelopes[3].Type = %q, want agent.run.replay.state", envelopes[3].Type)
	}
	approvalPayload, ok := envelopes[2].Payload.(map[string]any)
	if !ok {
		t.Fatalf("approval payload = %#v, want map", envelopes[2].Payload)
	}
	if approvalPayload["tool_call_id"] != "call_approval_1" {
		t.Fatalf("approvalPayload.tool_call_id = %v, want call_approval_1", approvalPayload["tool_call_id"])
	}
	if approvalPayload["tool_name"] != string(agents.ToolWriteFile) {
		t.Fatalf("approvalPayload.tool_name = %v, want %q", approvalPayload["tool_name"], agents.ToolWriteFile)
	}
	if approvalPayload["status"] != sqlite.ApprovalStateProposed {
		t.Fatalf("approvalPayload.status = %v, want %q", approvalPayload["status"], sqlite.ApprovalStateProposed)
	}
}

func TestService_OpenRunReplaysClarificationRequiredTerminalRun(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Replay clarification-required run")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Need planner clarification", sqlite.RolePlanner, config.DefaultPlannerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateHalted
	run.ErrorCode = "planner_clarification_required"
	run.ErrorMessage = "The run stopped because the planner asked for user clarification instead of producing actionable output."
	run.CompletedAt = &completedAt
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeAgentError, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","agent_id":"agent_planner_1","role":"planner","model":"`+run.Model+`","code":"planner_clarification_required","message":"The run stopped because the planner asked for user clarification instead of producing actionable output.","terminal":true,"occurred_at":"2026-03-24T12:00:01Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() agent_error error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeRunError, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","agent_id":"agent_planner_1","role":"planner","model":"`+run.Model+`","code":"planner_clarification_required","message":"The run stopped because the planner asked for user clarification instead of producing actionable output.","terminal":true,"occurred_at":"2026-03-24T12:00:02Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() run_error error = %v", err)
	}

	envelopes := make([]StreamEnvelope, 0, 2)
	_, err = service.OpenRun(ctx, OpenRunInput{SessionID: session.ID, RunID: run.ID}, func(envelope StreamEnvelope) error {
		envelopes = append(envelopes, envelope)
		return nil
	})
	if err != nil {
		t.Fatalf("OpenRun() error = %v", err)
	}
	if len(envelopes) != 4 {
		t.Fatalf("len(envelopes) = %d, want 4", len(envelopes))
	}
	if envelopes[0].Type != "agent.run.replay.state" || envelopes[1].Type != sqlite.EventTypeAgentError || envelopes[2].Type != sqlite.EventTypeRunError || envelopes[3].Type != "agent.run.replay.state" {
		t.Fatalf("envelopes = %#v, want replay state, agent_error, run_error, replay state", envelopes)
	}
	for index, envelope := range envelopes[1:3] {
		payload, ok := envelope.Payload.(map[string]any)
		if !ok {
			t.Fatalf("envelopes[%d].Payload = %#v, want map", index, envelope.Payload)
		}
		if payload["replay"] != true {
			t.Fatalf("envelopes[%d].payload.replay = %v, want true", index, payload["replay"])
		}
		if payload["code"] != "planner_clarification_required" {
			t.Fatalf("envelopes[%d].payload.code = %v, want planner_clarification_required", index, payload["code"])
		}
	}
}

func TestService_ReplayControlSeekReplaysUpToCursor(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Replay seek control")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Replay historical run", sqlite.RoleCoder, config.DefaultCoderModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateCompleted
	run.CompletedAt = &completedAt
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeStateChange, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","state":"running","occurred_at":"2026-03-24T12:00:00Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() first error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeAgentStateChanged, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","state":"completed","occurred_at":"2026-03-24T12:00:02Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() second error = %v", err)
	}

	if _, err := service.OpenRun(ctx, OpenRunInput{SessionID: session.ID, RunID: run.ID}, func(StreamEnvelope) error { return nil }); err != nil {
		t.Fatalf("OpenRun() error = %v", err)
	}

	envelopes := make([]StreamEnvelope, 0, 4)
	err = service.ReplayControl(ctx, ReplayControlInput{
		SessionID:  session.ID,
		RunID:      run.ID,
		Action:     ReplayActionSeek,
		CursorMS:   0,
		DirectUser: true,
	}, func(envelope StreamEnvelope) error {
		envelopes = append(envelopes, envelope)
		return nil
	})
	if err != nil {
		t.Fatalf("ReplayControl() error = %v", err)
	}
	if len(envelopes) != 3 {
		t.Fatalf("len(envelopes) = %d, want 3", len(envelopes))
	}
	if envelopes[0].Type != "agent.run.replay.state" || envelopes[1].Type != sqlite.EventTypeStateChange || envelopes[2].Type != "agent.run.replay.state" {
		t.Fatalf("envelopes = %#v, want replay.state, first event, replay.state", envelopes)
	}
	payload, ok := envelopes[1].Payload.(map[string]any)
	if !ok {
		t.Fatalf("event payload = %#v, want map", envelopes[1].Payload)
	}
	if payload["state"] != "running" {
		t.Fatalf("payload.state = %v, want running", payload["state"])
	}
}

func TestService_ReplayControlPlayStreamsEventsOverTime(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Replay play control")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Replay historical run", sqlite.RoleCoder, config.DefaultCoderModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateCompleted
	run.CompletedAt = &completedAt
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeStateChange, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","state":"running","occurred_at":"2026-03-24T12:00:00Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() first error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToken, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","text":"later","occurred_at":"2026-03-24T12:00:00.050Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() second error = %v", err)
	}

	if _, err := service.OpenRun(ctx, OpenRunInput{SessionID: session.ID, RunID: run.ID}, func(StreamEnvelope) error { return nil }); err != nil {
		t.Fatalf("OpenRun() error = %v", err)
	}
	if err := service.ReplayControl(ctx, ReplayControlInput{SessionID: session.ID, RunID: run.ID, Action: ReplayActionReset, DirectUser: true}, func(StreamEnvelope) error { return nil }); err != nil {
		t.Fatalf("ReplayControl(reset) error = %v", err)
	}

	envelopes := make(chan StreamEnvelope, 8)
	if err := service.ReplayControl(ctx, ReplayControlInput{SessionID: session.ID, RunID: run.ID, Action: ReplayActionPlay, DirectUser: true}, func(envelope StreamEnvelope) error {
		envelopes <- envelope
		return nil
	}); err != nil {
		t.Fatalf("ReplayControl(play) error = %v", err)
	}

	first := <-envelopes
	if first.Type != "agent.run.replay.state" {
		t.Fatalf("first envelope = %q, want agent.run.replay.state", first.Type)
	}
	if payload, ok := first.Payload.(map[string]any); !ok || payload["status"] != "playing" {
		t.Fatalf("first payload = %#v, want playing replay state", first.Payload)
	}
	select {
	case unexpected := <-envelopes:
		t.Fatalf("unexpected immediate replay frame = %#v", unexpected)
	case <-time.After(20 * time.Millisecond):
	}

	var sawToken bool
	deadline := time.After(250 * time.Millisecond)
	for !sawToken {
		select {
		case envelope := <-envelopes:
			if envelope.Type == sqlite.EventTypeToken {
				sawToken = true
			}
		case <-deadline:
			t.Fatal("timed out waiting for streamed replay token")
		}
	}
	finalState := readReplayStateByStatus(t, envelopes, "completed", 250*time.Millisecond)
	if payload, ok := finalState.Payload.(map[string]any); !ok || payload["status"] != "completed" {
		t.Fatalf("final payload = %#v, want completed replay state", finalState.Payload)
	}
}

func TestService_ReplayControlPauseStopsTimedPlayback(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Replay pause control")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Replay historical run", sqlite.RoleCoder, config.DefaultCoderModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateCompleted
	run.CompletedAt = &completedAt
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeStateChange, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","state":"running","occurred_at":"2026-03-24T12:00:00Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() first error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToken, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","text":"first","occurred_at":"2026-03-24T12:00:00.040Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() second error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToken, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","text":"second","occurred_at":"2026-03-24T12:00:00.180Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() third error = %v", err)
	}

	if _, err := service.OpenRun(ctx, OpenRunInput{SessionID: session.ID, RunID: run.ID}, func(StreamEnvelope) error { return nil }); err != nil {
		t.Fatalf("OpenRun() error = %v", err)
	}
	if err := service.ReplayControl(ctx, ReplayControlInput{SessionID: session.ID, RunID: run.ID, Action: ReplayActionReset, DirectUser: true}, func(StreamEnvelope) error { return nil }); err != nil {
		t.Fatalf("ReplayControl(reset) error = %v", err)
	}

	envelopes := make(chan StreamEnvelope, 12)
	if err := service.ReplayControl(ctx, ReplayControlInput{SessionID: session.ID, RunID: run.ID, Action: ReplayActionPlay, DirectUser: true}, func(envelope StreamEnvelope) error {
		envelopes <- envelope
		return nil
	}); err != nil {
		t.Fatalf("ReplayControl(play) error = %v", err)
	}
	_ = <-envelopes
	firstToken := readEnvelopeByType(t, envelopes, sqlite.EventTypeToken, 250*time.Millisecond)
	if payload, ok := firstToken.Payload.(map[string]any); !ok || payload["text"] != "first" {
		t.Fatalf("first token payload = %#v, want first", firstToken.Payload)
	}
	paused := make(chan StreamEnvelope, 2)
	if err := service.ReplayControl(ctx, ReplayControlInput{SessionID: session.ID, RunID: run.ID, Action: ReplayActionPause, DirectUser: true}, func(envelope StreamEnvelope) error {
		paused <- envelope
		return nil
	}); err != nil {
		t.Fatalf("ReplayControl(pause) error = %v", err)
	}
	pauseState := <-paused
	if payload, ok := pauseState.Payload.(map[string]any); !ok || payload["status"] != "paused" {
		t.Fatalf("pause payload = %#v, want paused", pauseState.Payload)
	}
	time.Sleep(220 * time.Millisecond)
	for {
		select {
		case envelope := <-envelopes:
			if envelope.Type == sqlite.EventTypeToken {
				payload := envelope.Payload.(map[string]any)
				if payload["text"] == "second" {
					t.Fatal("second token was emitted after pause")
				}
			}
		default:
			return
		}
	}
}

func TestService_ReplayControlPlayRestartsCompletedReplayFromBeginning(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Replay restart from completed")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Replay historical run", sqlite.RoleCoder, config.DefaultCoderModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateCompleted
	run.CompletedAt = &completedAt
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeStateChange, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","state":"running","occurred_at":"2026-03-24T12:00:00Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() first error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToken, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","text":"replayed","occurred_at":"2026-03-24T12:00:00.050Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() second error = %v", err)
	}

	if _, err := service.OpenRun(ctx, OpenRunInput{SessionID: session.ID, RunID: run.ID}, func(StreamEnvelope) error { return nil }); err != nil {
		t.Fatalf("OpenRun() error = %v", err)
	}
	if err := service.ReplayControl(ctx, ReplayControlInput{SessionID: session.ID, RunID: run.ID, Action: ReplayActionSetSpeed, Speed: 5, DirectUser: true}, func(StreamEnvelope) error { return nil }); err != nil {
		t.Fatalf("ReplayControl(set_speed) error = %v", err)
	}

	envelopes := make(chan StreamEnvelope, 8)
	if err := service.ReplayControl(ctx, ReplayControlInput{SessionID: session.ID, RunID: run.ID, Action: ReplayActionPlay, DirectUser: true}, func(envelope StreamEnvelope) error {
		envelopes <- envelope
		return nil
	}); err != nil {
		t.Fatalf("ReplayControl(play) error = %v", err)
	}

	first := <-envelopes
	firstPayload, ok := first.Payload.(map[string]any)
	if first.Type != "agent.run.replay.state" || !ok {
		t.Fatalf("first envelope = %#v, want replay state payload", first)
	}
	if firstPayload["status"] != "seeking" {
		t.Fatalf("first status = %v, want seeking", firstPayload["status"])
	}
	if firstPayload["cursor_ms"] != 0 {
		t.Fatalf("first cursor_ms = %v, want 0", firstPayload["cursor_ms"])
	}
	if firstPayload["speed"] != 5.0 {
		t.Fatalf("first speed = %v, want 5", firstPayload["speed"])
	}

	replayedState := readEnvelopeByType(t, envelopes, sqlite.EventTypeStateChange, 100*time.Millisecond)
	if payload, ok := replayedState.Payload.(map[string]any); !ok || payload["state"] != "running" {
		t.Fatalf("state payload = %#v, want running", replayedState.Payload)
	}

	pausedState := readReplayStateByStatus(t, envelopes, "paused", 100*time.Millisecond)
	if payload, ok := pausedState.Payload.(map[string]any); !ok || payload["cursor_ms"] != 0 {
		t.Fatalf("paused payload = %#v, want cursor_ms 0", pausedState.Payload)
	}

	playingState := readReplayStateByStatus(t, envelopes, "playing", 100*time.Millisecond)
	if payload, ok := playingState.Payload.(map[string]any); !ok || payload["cursor_ms"] != 0 {
		t.Fatalf("playing payload = %#v, want cursor_ms 0", playingState.Payload)
	}

	token := readEnvelopeByType(t, envelopes, sqlite.EventTypeToken, 100*time.Millisecond)
	if payload, ok := token.Payload.(map[string]any); !ok || payload["text"] != "replayed" {
		t.Fatalf("token payload = %#v, want replayed", token.Payload)
	}
}

func TestBuildReplayTimelineAddsSyntheticSpacingForCoarseTimestamps(t *testing.T) {
	timeline := buildReplayTimeline("run_coarse", []sqlite.AgentRunEvent{
		{
			Sequence:    1,
			EventType:   sqlite.EventTypeStateChange,
			PayloadJSON: `{"occurred_at":"2026-03-24T12:00:00Z"}`,
			CreatedAt:   time.Date(2026, time.March, 24, 12, 0, 0, 0, time.UTC),
		},
		{
			Sequence:    2,
			EventType:   sqlite.EventTypeToken,
			PayloadJSON: `{"occurred_at":"2026-03-24T12:00:00Z"}`,
			CreatedAt:   time.Date(2026, time.March, 24, 12, 0, 0, 0, time.UTC),
		},
	})

	if len(timeline.Events) != 2 {
		t.Fatalf("len(timeline.Events) = %d, want 2", len(timeline.Events))
	}
	if timeline.Events[0].RelativeMS != 0 {
		t.Fatalf("timeline.Events[0].RelativeMS = %d, want 0", timeline.Events[0].RelativeMS)
	}
	if timeline.Events[1].RelativeMS != replaySyntheticStepMS {
		t.Fatalf("timeline.Events[1].RelativeMS = %d, want %d", timeline.Events[1].RelativeMS, replaySyntheticStepMS)
	}
	if timeline.DurationMS != replaySyntheticStepMS {
		t.Fatalf("timeline.DurationMS = %d, want %d", timeline.DurationMS, replaySyntheticStepMS)
	}
}

func TestService_ReplayControlPlayHonorsSpeedForRecordedEvents(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Replay speed honors recorded timing")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Measure replay speed", sqlite.RoleCoder, config.DefaultCoderModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	if err := service.emitStateChange(ctx, run, sqlite.RunStateThinking, "thinking", false, func(StreamEnvelope) error { return nil }); err != nil {
		t.Fatalf("emitStateChange() error = %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if err := service.emitToken(ctx, run.ID, "timed token", func(StreamEnvelope) error { return nil }); err != nil {
		t.Fatalf("emitToken() error = %v", err)
	}
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateCompleted
	run.CompletedAt = &completedAt
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}

	if _, err := service.OpenRun(ctx, OpenRunInput{SessionID: session.ID, RunID: run.ID}, func(StreamEnvelope) error { return nil }); err != nil {
		t.Fatalf("OpenRun() error = %v", err)
	}

	measureTokenDelay := func(speed float64) time.Duration {
		t.Helper()
		if err := service.ReplayControl(ctx, ReplayControlInput{SessionID: session.ID, RunID: run.ID, Action: ReplayActionSetSpeed, Speed: speed, DirectUser: true}, func(StreamEnvelope) error { return nil }); err != nil {
			t.Fatalf("ReplayControl(set_speed=%v) error = %v", speed, err)
		}
		envelopes := make(chan StreamEnvelope, 8)
		startedAt := time.Now()
		if err := service.ReplayControl(ctx, ReplayControlInput{SessionID: session.ID, RunID: run.ID, Action: ReplayActionPlay, DirectUser: true}, func(envelope StreamEnvelope) error {
			envelopes <- envelope
			return nil
		}); err != nil {
			t.Fatalf("ReplayControl(play speed=%v) error = %v", speed, err)
		}
		readEnvelopeByType(t, envelopes, sqlite.EventTypeToken, 2*time.Second)
		return time.Since(startedAt)
	}

	fastDelay := measureTokenDelay(5)
	normalDelay := measureTokenDelay(1)
	slowDelay := measureTokenDelay(0.5)

	if fastDelay >= 150*time.Millisecond {
		t.Fatalf("fastDelay = %v, want under 150ms", fastDelay)
	}
	if normalDelay <= 150*time.Millisecond || normalDelay >= 450*time.Millisecond {
		t.Fatalf("normalDelay = %v, want between 150ms and 450ms", normalDelay)
	}
	if slowDelay <= 250*time.Millisecond {
		t.Fatalf("slowDelay = %v, want over 250ms", slowDelay)
	}
	if normalDelay <= fastDelay*2 {
		t.Fatalf("normalDelay = %v, fastDelay = %v, want 1x replay to take meaningfully longer than 5x", normalDelay, fastDelay)
	}
	if slowDelay <= normalDelay {
		t.Fatalf("slowDelay = %v, normalDelay = %v, want 0.5x replay to take longer than 1x", slowDelay, normalDelay)
	}
	if slowDelay <= fastDelay*3 {
		t.Fatalf("slowDelay = %v, fastDelay = %v, want slow replay to take meaningfully longer", slowDelay, fastDelay)
	}
}

func TestService_ReplayControlPlayStreamsProgressBetweenFrames(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Replay progress updates")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Observe replay cursor progress", sqlite.RoleCoder, config.DefaultCoderModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateCompleted
	run.CompletedAt = &completedAt
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeStateChange, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","state":"running","occurred_at":"2026-03-24T12:00:00Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() first error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToken, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","text":"later","occurred_at":"2026-03-24T12:00:00.300Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() second error = %v", err)
	}

	if _, err := service.OpenRun(ctx, OpenRunInput{SessionID: session.ID, RunID: run.ID}, func(StreamEnvelope) error { return nil }); err != nil {
		t.Fatalf("OpenRun() error = %v", err)
	}

	envelopes := make(chan StreamEnvelope, 32)
	if err := service.ReplayControl(ctx, ReplayControlInput{SessionID: session.ID, RunID: run.ID, Action: ReplayActionPlay, DirectUser: true}, func(envelope StreamEnvelope) error {
		envelopes <- envelope
		return nil
	}); err != nil {
		t.Fatalf("ReplayControl(play) error = %v", err)
	}

	deadline := time.After(400 * time.Millisecond)
	for {
		select {
		case envelope := <-envelopes:
			if envelope.Type != "agent.run.replay.state" {
				continue
			}
			payload, ok := envelope.Payload.(map[string]any)
			if !ok {
				continue
			}
			status, _ := payload["status"].(string)
			cursor, cursorOK := payload["cursor_ms"].(int)
			if !cursorOK {
				if raw, castOK := payload["cursor_ms"].(float64); castOK {
					cursor = int(raw)
				}
			}
			if status == "playing" && cursor > 0 && cursor < 300 {
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for intermediate replay progress state")
		}
	}
}

func TestDeriveParticipantsCountsRecordedAgentsWithoutInflatingFromSearchText(t *testing.T) {
	run := sqlite.AgentRun{
		Role:  sqlite.RoleCoder,
		Model: config.DefaultCoderModel,
	}
	executions := []sqlite.AgentExecution{
		{ID: "agent_planner_1", Role: sqlite.RolePlanner, Model: config.DefaultPlannerModel},
		{ID: "agent_coder_1", Role: sqlite.RoleCoder, Model: config.DefaultCoderModel},
	}
	events := []sqlite.AgentRunEvent{
		{
			Role:        sqlite.RolePlanner,
			Model:       config.DefaultPlannerModel,
			PayloadJSON: `{"agent_id":"agent_planner_1"}`,
		},
		{
			Role:        sqlite.RoleCoder,
			Model:       config.DefaultCoderModel,
			PayloadJSON: `{"agent_id":"agent_coder_1"}`,
		},
	}

	participants, agentCount := deriveParticipants(run, executions, events)

	if agentCount != 2 {
		t.Fatalf("agentCount = %d, want 2", agentCount)
	}
	if strings.Contains(participants, "agent_planner_1") || strings.Contains(participants, "agent_coder_1") {
		t.Fatalf("participants = %q, want role/model search text without agent ids", participants)
	}
}

func TestService_ExportRunHistoryRejectsNonDirectRequests(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Replay export governance")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Export historical run", sqlite.RoleReviewer, config.DefaultReviewerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	if _, err := service.ExportRunHistory(ctx, RunHistoryExportRequest{SessionID: session.ID, RunID: run.ID, DirectUser: false}); err == nil {
		t.Fatal("ExportRunHistory() error = nil, want direct-user rejection")
	}
}

func TestService_ExportRunHistoryWritesMarkdown(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Replay export markdown")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Review approval flow", sqlite.RoleReviewer, config.DefaultReviewerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateCompleted
	run.CompletedAt = &completedAt
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToken, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","text":"preserved transcript","occurred_at":"2026-03-24T12:00:01Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() token error = %v", err)
	}
	inputPreviewJSON, err := json.Marshal(map[string]any{
		"path": "README.md",
		"diff_preview": map[string]any{
			"target_path":       "README.md",
			"original_content":  "before\n",
			"proposed_content":  "after\n",
			"base_content_hash": "sha256:abc",
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if _, err := store.CreateApprovalRequest(ctx, sqlite.ApprovalRequest{
		SessionID:        session.ID,
		RunID:            run.ID,
		ToolCallID:       "call_export_1",
		ToolName:         string(agents.ToolWriteFile),
		Role:             sqlite.RoleReviewer,
		Model:            config.DefaultReviewerModel,
		InputPreviewJSON: string(inputPreviewJSON),
		Message:          "Relay needs approval before it can write files inside the configured project root.",
		State:            sqlite.ApprovalStateApplied,
		OccurredAt:       completedAt,
	}); err != nil {
		t.Fatalf("CreateApprovalRequest() error = %v", err)
	}

	result, err := service.ExportRunHistory(ctx, RunHistoryExportRequest{
		SessionID:   session.ID,
		RunID:       run.ID,
		DirectUser:  true,
		RequestedAt: time.Date(2026, time.March, 24, 12, 3, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ExportRunHistory() error = %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("result.Status = %q, want completed", result.Status)
	}
	if filepath.Dir(result.ExportPath) != filepath.Join(paths.ConfigDir, "exports") {
		t.Fatalf("result.ExportPath = %q, want exports dir inside %q", result.ExportPath, paths.ConfigDir)
	}
	body, err := os.ReadFile(result.ExportPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(body)
	for _, expected := range []string{"# Review approval flow", "## Timeline", "preserved transcript", "## Changes", "README.md", "before", "after"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("export content missing %q in %q", expected, content)
		}
	}
}

func TestService_ExportRunHistoryAllowsRepeatedExportsForSameRun(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Replay repeated export markdown")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Review approval flow", sqlite.RoleReviewer, config.DefaultReviewerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateCompleted
	run.CompletedAt = &completedAt
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToken, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","text":"preserved transcript","occurred_at":"2026-03-24T12:00:01Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() token error = %v", err)
	}

	firstResult, err := service.ExportRunHistory(ctx, RunHistoryExportRequest{
		SessionID:   session.ID,
		RunID:       run.ID,
		DirectUser:  true,
		RequestedAt: time.Date(2026, time.March, 24, 12, 3, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("first ExportRunHistory() error = %v", err)
	}
	secondResult, err := service.ExportRunHistory(ctx, RunHistoryExportRequest{
		SessionID:   session.ID,
		RunID:       run.ID,
		DirectUser:  true,
		RequestedAt: time.Date(2026, time.March, 24, 12, 3, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("second ExportRunHistory() error = %v", err)
	}
	if firstResult.ExportPath == secondResult.ExportPath {
		t.Fatalf("second export path = %q, want unique path distinct from %q", secondResult.ExportPath, firstResult.ExportPath)
	}
	if _, err := os.Stat(secondResult.ExportPath); err != nil {
		t.Fatalf("os.Stat(secondResult.ExportPath) error = %v", err)
	}
}

func TestBuildReplayCheckpoints(t *testing.T) {
	frames := []replayFrame{
		{RelativeMS: 0, Envelope: StreamEnvelope{Type: "agent_spawned", Payload: map[string]any{"sequence": int64(1)}}},
		{RelativeMS: 40, Envelope: StreamEnvelope{Type: "token", Payload: map[string]any{"sequence": int64(2)}}},
		{RelativeMS: 90, Envelope: StreamEnvelope{Type: "tool_call", Payload: map[string]any{"sequence": int64(3)}}},
		{RelativeMS: 140, Envelope: StreamEnvelope{Type: "tool_result", Payload: map[string]any{"sequence": int64(4)}}},
	}

	checkpoints := buildReplayCheckpoints("run_history_1", frames)
	if len(checkpoints) != 2 {
		t.Fatalf("len(checkpoints) = %d, want 2", len(checkpoints))
	}
	if checkpoints[0].CheckpointMS != 90 || checkpoints[0].FrameCount != 3 {
		t.Fatalf("first checkpoint = %+v, want checkpoint at 90ms with 3 frames", checkpoints[0])
	}
	if checkpoints[1].CheckpointMS != 140 || checkpoints[1].LastSequence != 4 {
		t.Fatalf("final checkpoint = %+v, want final sequence 4 at 140ms", checkpoints[1])
	}
	checkpoint, ok := replayCheckpointForCursor(checkpoints, 100)
	if !ok {
		t.Fatal("replayCheckpointForCursor() ok = false, want true")
	}
	if checkpoint.FrameCount != 3 {
		t.Fatalf("checkpoint.FrameCount = %d, want 3", checkpoint.FrameCount)
	}
}

func readEnvelopeByType(t *testing.T, envelopes <-chan StreamEnvelope, envelopeType string, timeout time.Duration) StreamEnvelope {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case envelope := <-envelopes:
			if envelope.Type == envelopeType {
				return envelope
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %q envelope", envelopeType)
		}
	}
}

func readReplayStateByStatus(t *testing.T, envelopes <-chan StreamEnvelope, status string, timeout time.Duration) StreamEnvelope {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case envelope := <-envelopes:
			if envelope.Type != "agent.run.replay.state" {
				continue
			}
			payload, ok := envelope.Payload.(map[string]any)
			if ok && payload["status"] == status {
				return envelope
			}
		case <-deadline:
			t.Fatalf("timed out waiting for replay state %q", status)
		}
	}
}

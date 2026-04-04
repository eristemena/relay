package workspace

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/eristemena/relay/internal/storage/sqlite"
)

type ReplayAction string

const (
	ReplayActionPlay     ReplayAction = "play"
	ReplayActionPause    ReplayAction = "pause"
	ReplayActionSeek     ReplayAction = "seek"
	ReplayActionSetSpeed ReplayAction = "set_speed"
	ReplayActionReset    ReplayAction = "reset"
)

const replayProgressTick = 50 * time.Millisecond

type ReplayControlInput struct {
	SessionID string
	RunID     string
	Action    ReplayAction
	CursorMS  int
	Speed     float64
	DirectUser bool
}

type ReplayState struct {
	SessionID         string    `json:"session_id"`
	RunID             string    `json:"run_id"`
	Status            string    `json:"status"`
	CursorMS          int       `json:"cursor_ms"`
	DurationMS        int       `json:"duration_ms"`
	Speed             float64   `json:"speed"`
	SelectedTimestamp time.Time `json:"selected_timestamp,omitempty"`
}

type replayEvent struct {
	Sequence   int64
	OccurredAt time.Time
	Type       string
	Payload    map[string]any
	RelativeMS int
}

type replayTimeline struct {
	RunID       string
	DurationMS  int
	Events      []replayEvent
	Checkpoints []ReplayCheckpoint
}

func replaySessionKey(sessionID string, runID string) string {
	return strings.TrimSpace(sessionID) + ":" + strings.TrimSpace(runID)
}

func (s *Service) storeReplaySession(sessionID string, runID string, timeline replayTimeline, approvals []sqlite.ApprovalRequest, session ReplaySession) {
	runtime := replaySessionRuntime{
		session:   session,
		timeline:  timeline,
		approvals: approvals,
	}
	runtime.frames = replayFrames(runtime)
	runtime.timeline.Checkpoints = buildReplayCheckpoints(runID, runtime.frames)
	s.mu.Lock()
	defer s.mu.Unlock()
	key := replaySessionKey(sessionID, runID)
	if existing, ok := s.replaySessions[key]; ok && existing.playbackCancel != nil {
		existing.playbackCancel()
	}
	s.replaySessions[replaySessionKey(sessionID, runID)] = runtime
}

func (s *Service) loadReplaySession(sessionID string, runID string) (replaySessionRuntime, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	runtime, ok := s.replaySessions[replaySessionKey(sessionID, runID)]
	return runtime, ok
}

func (s *Service) updateReplaySession(runtime replaySessionRuntime) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replaySessions[replaySessionKey(runtime.session.WorkspaceID, runtime.session.RunID)] = runtime
}

func (s *Service) stopReplayPlayback(sessionID string, runID string) (replaySessionRuntime, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := replaySessionKey(sessionID, runID)
	runtime, ok := s.replaySessions[key]
	if !ok {
		return replaySessionRuntime{}, false
	}
	if runtime.playbackCancel != nil {
		runtime.playbackCancel()
		runtime.playbackCancel = nil
	}
	runtime.playbackToken++
	s.replaySessions[key] = runtime
	return runtime, true
}

func (s *Service) beginReplayPlayback(sessionID string, runID string) (replaySessionRuntime, context.Context, int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := replaySessionKey(sessionID, runID)
	runtime, ok := s.replaySessions[key]
	if !ok {
		return replaySessionRuntime{}, nil, 0, false
	}
	if runtime.playbackCancel != nil {
		runtime.playbackCancel()
	}
	runtime.playbackToken++
	token := runtime.playbackToken
	playCtx, cancel := context.WithCancel(context.Background())
	runtime.playbackCancel = cancel
	runtime.session.Status = "playing"
	s.replaySessions[key] = runtime
	return runtime, playCtx, token, true
}

func (s *Service) updateReplayProgress(sessionID string, runID string, token int64, cursorMS int, status string, clearPlayback bool) (replaySessionRuntime, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := replaySessionKey(sessionID, runID)
	runtime, ok := s.replaySessions[key]
	if !ok || runtime.playbackToken != token {
		return replaySessionRuntime{}, false
	}
	runtime.session.CursorMS = cursorMS
	runtime.session.Status = status
	if clearPlayback {
		runtime.playbackCancel = nil
	}
	s.replaySessions[key] = runtime
	return runtime, true
}

func (s *Service) ReplayControl(ctx context.Context, input ReplayControlInput, emit func(StreamEnvelope) error) error {
	_ = ctx
	if !input.DirectUser {
		return fmt.Errorf("replay control requires a direct user request")
	}
	runtime, ok := s.loadReplaySession(input.SessionID, input.RunID)
	if !ok {
		return fmt.Errorf("replay session for run %s is not initialized", input.RunID)
	}

	switch input.Action {
	case ReplayActionPause:
		runtime, _ = s.stopReplayPlayback(input.SessionID, input.RunID)
		runtime.session.Status = "paused"
		s.updateReplaySession(runtime)
		return emit(StreamEnvelope{Type: "agent.run.replay.state", Payload: replayStatePayload(input.SessionID, input.RunID, "paused", runtime.session.CursorMS, runtime.session.DurationMS, runtime.session.Speed, replayTimestampForCursor(runtime.timeline, runtime.session.CursorMS), len(runtime.timeline.Events) > 0)})
	case ReplayActionReset:
		runtime, _ = s.stopReplayPlayback(input.SessionID, input.RunID)
		return s.replayToCursor(input, runtime, 0, "paused", emit)
	case ReplayActionSeek:
		runtime, _ = s.stopReplayPlayback(input.SessionID, input.RunID)
		if input.CursorMS < 0 {
			input.CursorMS = 0
		}
		if input.CursorMS > runtime.session.DurationMS {
			input.CursorMS = runtime.session.DurationMS
		}
		return s.replayToCursor(input, runtime, input.CursorMS, "paused", emit)
	case ReplayActionPlay:
		if runtime.session.Status == "completed" || runtime.session.CursorMS >= runtime.session.DurationMS {
			runtime, _ = s.stopReplayPlayback(input.SessionID, input.RunID)
			if err := s.replayToCursor(input, runtime, 0, "paused", emit); err != nil {
				return err
			}
			runtime, ok = s.loadReplaySession(input.SessionID, input.RunID)
			if !ok {
				return fmt.Errorf("replay session for run %s is not initialized", input.RunID)
			}
		}
		runtime, playCtx, token, ok := s.beginReplayPlayback(input.SessionID, input.RunID)
		if !ok {
			return fmt.Errorf("replay session for run %s is not initialized", input.RunID)
		}
		if err := emit(StreamEnvelope{Type: "agent.run.replay.state", Payload: replayStatePayload(input.SessionID, input.RunID, "playing", runtime.session.CursorMS, runtime.session.DurationMS, runtime.session.Speed, replayTimestampForCursor(runtime.timeline, runtime.session.CursorMS), len(runtime.timeline.Events) > 0)}); err != nil {
			return err
		}
		go s.streamReplayPlayback(playCtx, input.SessionID, input.RunID, token, runtime, emit)
		return nil
	case ReplayActionSetSpeed:
		runtime, _ = s.stopReplayPlayback(input.SessionID, input.RunID)
		if input.Speed <= 0 {
			return fmt.Errorf("replay speed must be positive")
		}
		runtime.session.Speed = input.Speed
		runtime.session.Status = "paused"
		s.updateReplaySession(runtime)
		return emit(StreamEnvelope{Type: "agent.run.replay.state", Payload: replayStatePayload(input.SessionID, input.RunID, "paused", runtime.session.CursorMS, runtime.session.DurationMS, runtime.session.Speed, replayTimestampForCursor(runtime.timeline, runtime.session.CursorMS), len(runtime.timeline.Events) > 0)})
	default:
		return fmt.Errorf("unsupported replay action %q", input.Action)
	}
}

func (s *Service) replayToCursor(input ReplayControlInput, runtime replaySessionRuntime, cursorMS int, terminalStatus string, emit func(StreamEnvelope) error) error {
	if err := emit(StreamEnvelope{Type: "agent.run.replay.state", Payload: replayStatePayload(input.SessionID, input.RunID, "seeking", cursorMS, runtime.session.DurationMS, runtime.session.Speed, replayTimestampForCursor(runtime.timeline, cursorMS), len(runtime.timeline.Events) > 0)}); err != nil {
		return err
	}
	startIndex := 0
	if checkpoint, ok := replayCheckpointForCursor(runtime.timeline.Checkpoints, cursorMS); ok {
		for _, envelope := range checkpoint.Envelopes {
			if err := emit(cloneReplayEnvelope(envelope)); err != nil {
				return err
			}
		}
		startIndex = checkpoint.FrameCount
	}
	for _, frame := range runtime.frames[startIndex:] {
		if frame.RelativeMS > cursorMS {
			break
		}
		if err := emit(cloneReplayEnvelope(frame.Envelope)); err != nil {
			return err
		}
	}
	runtime.session.CursorMS = cursorMS
	runtime.session.Status = terminalStatus
	s.updateReplaySession(runtime)
	return emit(StreamEnvelope{Type: "agent.run.replay.state", Payload: replayStatePayload(input.SessionID, input.RunID, terminalStatus, cursorMS, runtime.session.DurationMS, runtime.session.Speed, replayTimestampForCursor(runtime.timeline, cursorMS), len(runtime.timeline.Events) > 0)})
}

type replayFrame struct {
	RelativeMS int
	Envelope   StreamEnvelope
}

func replayFrames(runtime replaySessionRuntime) []replayFrame {
	frames := make([]replayFrame, 0, len(runtime.timeline.Events)+len(runtime.approvals))
	for _, event := range runtime.timeline.Events {
		frames = append(frames, replayFrame{
			RelativeMS: event.RelativeMS,
			Envelope:   StreamEnvelope{Type: event.Type, Payload: cloneReplayPayload(event)},
		})
	}
	for _, approval := range runtime.approvals {
		summary, err := approvalSummaryFromRecord(approval)
		if err != nil {
			continue
		}
		frames = append(frames, replayFrame{
			RelativeMS: replayApprovalRelativeMS(runtime.timeline, approval.OccurredAt),
			Envelope:   StreamEnvelope{Type: "approval_request", Payload: approvalPayload(summary)},
		})
	}
	sort.SliceStable(frames, func(i int, j int) bool {
		return frames[i].RelativeMS < frames[j].RelativeMS
	})
	return frames
}

func replayApprovalRelativeMS(timeline replayTimeline, occurredAt time.Time) int {
	if len(timeline.Events) == 0 {
		return 0
	}
	relativeMS := int(occurredAt.Sub(timeline.Events[0].OccurredAt).Milliseconds())
	if relativeMS < 0 {
		return 0
	}
	return relativeMS
}

func (s *Service) streamReplayPlayback(ctx context.Context, sessionID string, runID string, token int64, runtime replaySessionRuntime, emit func(StreamEnvelope) error) {
	frames := runtime.frames
	if len(frames) == 0 {
		frames = replayFrames(runtime)
	}
	lastCursor := runtime.session.CursorMS
	speed := runtime.session.Speed
	if speed <= 0 {
		speed = 1
	}
	for _, frame := range frames {
		if frame.RelativeMS <= lastCursor {
			continue
		}
		waitMS := frame.RelativeMS - lastCursor
		if waitMS > 0 {
			delay := time.Duration(float64(waitMS)/speed*float64(time.Millisecond))
			if delay > 0 {
				timer := time.NewTimer(delay)
				ticker := time.NewTicker(replayProgressTick)
				segmentStartedAt := time.Now()
				for {
					select {
					case <-ctx.Done():
						ticker.Stop()
						if !timer.Stop() {
							select {
							case <-timer.C:
							default:
							}
						}
						return
					case <-ticker.C:
						elapsed := time.Since(segmentStartedAt)
						if elapsed >= delay {
							continue
						}
						progressRatio := float64(elapsed) / float64(delay)
						cursorMS := lastCursor + int(math.Round(float64(waitMS)*progressRatio))
						if cursorMS <= runtime.session.CursorMS {
							continue
						}
						if cursorMS > frame.RelativeMS {
							cursorMS = frame.RelativeMS
						}
						updated, ok := s.updateReplayProgress(sessionID, runID, token, cursorMS, "playing", false)
						if !ok {
							ticker.Stop()
							if !timer.Stop() {
								select {
								case <-timer.C:
								default:
								}
							}
							return
						}
						runtime = updated
						if err := emit(StreamEnvelope{Type: "agent.run.replay.state", Payload: replayStatePayload(sessionID, runID, "playing", runtime.session.CursorMS, runtime.session.DurationMS, runtime.session.Speed, replayTimestampForCursor(runtime.timeline, runtime.session.CursorMS), len(runtime.timeline.Events) > 0)}); err != nil {
							ticker.Stop()
							if !timer.Stop() {
								select {
								case <-timer.C:
								default:
								}
							}
							if s.logger != nil {
								s.logger.Debug("replay playback progress emit failed", "run_id", runID, "error", err)
							}
							return
						}
					case <-timer.C:
						ticker.Stop()
						goto playbackReady
					}
				}
			}
		}
	playbackReady:
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := emit(cloneReplayEnvelope(frame.Envelope)); err != nil {
			if s.logger != nil {
				s.logger.Debug("replay playback emit failed", "run_id", runID, "error", err)
			}
			return
		}
		updated, ok := s.updateReplayProgress(sessionID, runID, token, frame.RelativeMS, "playing", false)
		if !ok {
			return
		}
		runtime = updated
		lastCursor = frame.RelativeMS
		if err := emit(StreamEnvelope{Type: "agent.run.replay.state", Payload: replayStatePayload(sessionID, runID, "playing", runtime.session.CursorMS, runtime.session.DurationMS, runtime.session.Speed, replayTimestampForCursor(runtime.timeline, runtime.session.CursorMS), len(runtime.timeline.Events) > 0)}); err != nil {
			if s.logger != nil {
				s.logger.Debug("replay playback progress emit failed", "run_id", runID, "error", err)
			}
			return
		}
	}
	updated, ok := s.updateReplayProgress(sessionID, runID, token, runtime.session.DurationMS, "completed", true)
	if !ok {
		return
	}
	_ = emit(StreamEnvelope{Type: "agent.run.replay.state", Payload: replayStatePayload(sessionID, runID, "completed", updated.session.CursorMS, updated.session.DurationMS, updated.session.Speed, replayTimestampForCursor(updated.timeline, updated.session.CursorMS), len(updated.timeline.Events) > 0)})
}

func replayTimestampForCursor(timeline replayTimeline, cursorMS int) time.Time {
	for index := len(timeline.Events) - 1; index >= 0; index-- {
		if timeline.Events[index].RelativeMS <= cursorMS {
			return timeline.Events[index].OccurredAt
		}
	}
	if len(timeline.Events) > 0 {
		return timeline.Events[0].OccurredAt
	}
	return time.Time{}
}
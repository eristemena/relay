package workspace

const replayCheckpointFrameInterval = 3

type ReplayCheckpoint struct {
	RunID          string
	CheckpointMS   int
	LastSequence   int64
	FrameCount     int
	Envelopes      []StreamEnvelope
	CanvasDocument any
	Transcript     map[string]string
	Approvals      map[string]any
	Summary        any
}

type ReplaySession struct {
	SessionID   string
	WorkspaceID string
	RunID       string
	Status      string
	Speed       float64
	CursorMS    int
	DurationMS  int
}

func buildReplayCheckpoints(runID string, frames []replayFrame) []ReplayCheckpoint {
	if len(frames) == 0 {
		return nil
	}
	checkpoints := make([]ReplayCheckpoint, 0, len(frames)/replayCheckpointFrameInterval+1)
	for index, frame := range frames {
		isIntervalCheckpoint := (index+1)%replayCheckpointFrameInterval == 0
		isFinalCheckpoint := index == len(frames)-1
		if !isIntervalCheckpoint && !isFinalCheckpoint {
			continue
		}
		checkpoints = append(checkpoints, ReplayCheckpoint{
			RunID:        runID,
			CheckpointMS: frame.RelativeMS,
			LastSequence: replayLastSequence(frames[:index+1]),
			FrameCount:   index + 1,
			Envelopes:    cloneReplayEnvelopes(frames[:index+1]),
		})
	}
	return checkpoints
}

func replayCheckpointForCursor(checkpoints []ReplayCheckpoint, cursorMS int) (ReplayCheckpoint, bool) {
	var selected ReplayCheckpoint
	for _, checkpoint := range checkpoints {
		if checkpoint.CheckpointMS > cursorMS {
			break
		}
		selected = checkpoint
	}
	if selected.FrameCount == 0 {
		return ReplayCheckpoint{}, false
	}
	return selected, true
}

func cloneReplayEnvelopes(frames []replayFrame) []StreamEnvelope {
	envelopes := make([]StreamEnvelope, 0, len(frames))
	for _, frame := range frames {
		envelopes = append(envelopes, cloneReplayEnvelope(frame.Envelope))
	}
	return envelopes
}

func cloneReplayEnvelope(envelope StreamEnvelope) StreamEnvelope {
	cloned := StreamEnvelope{Type: envelope.Type}
	if payload, ok := envelope.Payload.(map[string]any); ok {
		payloadClone := make(map[string]any, len(payload))
		for key, value := range payload {
			payloadClone[key] = value
		}
		cloned.Payload = payloadClone
		return cloned
	}
	cloned.Payload = envelope.Payload
	return cloned
}

func replayLastSequence(frames []replayFrame) int64 {
	for index := len(frames) - 1; index >= 0; index-- {
		if payload, ok := frames[index].Envelope.Payload.(map[string]any); ok {
			if sequence, ok := payload["sequence"].(int64); ok {
				return sequence
			}
			if sequence, ok := payload["sequence"].(int); ok {
				return int64(sequence)
			}
			if sequence, ok := payload["sequence"].(float64); ok {
				return int64(sequence)
			}
		}
	}
	return 0
}
package workspace

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/eristemena/relay/internal/config"
	"github.com/eristemena/relay/internal/storage/sqlite"
)

type RepositoryTreeRequestInput struct {
	SessionID string
	RunID     string
}

type TouchedFileSummary struct {
	RunID     string `json:"run_id"`
	AgentID   string `json:"agent_id"`
	FilePath  string `json:"file_path"`
	TouchType string `json:"touch_type"`
}

type RepositoryTreeResult struct {
	SessionID      string               `json:"session_id,omitempty"`
	RunID          string               `json:"run_id,omitempty"`
	RepositoryRoot string               `json:"repository_root,omitempty"`
	Status         string               `json:"status"`
	Message        string               `json:"message,omitempty"`
	Paths          []string             `json:"paths,omitempty"`
	TouchedFiles   []TouchedFileSummary `json:"touched_files,omitempty"`
}

func (s *Service) GetRepositoryTree(ctx context.Context, input RepositoryTreeRequestInput) (RepositoryTreeResult, error) {
	cfg, _, err := config.Load(s.paths)
	if err != nil {
		return RepositoryTreeResult{}, err
	}
	s.syncRepositoryTree(ctx, cfg)

	connected := connectedRepositorySummary(cfg.SafePreferences())
	if connected.Status != "connected" || strings.TrimSpace(connected.Path) == "" {
		message := strings.TrimSpace(connected.Message)
		if message == "" {
			message = "Relay could not load the repository tree because no valid repository is connected."
		}
		return RepositoryTreeResult{}, fmt.Errorf("%s", message)
	}

	entry := s.currentRepositoryTree(connected)
	if entry.Status == repositoryTreeStatusError {
		message := strings.TrimSpace(entry.ErrorMessage)
		if message == "" {
			message = "Relay could not build the repository tree for the connected repository."
		}
		return RepositoryTreeResult{}, fmt.Errorf("%s", message)
	}

	result := RepositoryTreeResult{
		SessionID:      input.SessionID,
		RunID:          input.RunID,
		RepositoryRoot: connected.Path,
		Status:         repositoryTreeStatusReady,
		Message:        "Repository tree is ready.",
		Paths:          append([]string(nil), entry.Snapshot.Paths...),
		TouchedFiles:   []TouchedFileSummary{},
	}
	if strings.TrimSpace(input.RunID) == "" {
		return result, nil
	}

	run, err := s.store.GetAgentRun(ctx, input.RunID)
	if err != nil {
		return RepositoryTreeResult{}, err
	}
	if strings.TrimSpace(input.SessionID) != "" && run.SessionID != input.SessionID {
		return RepositoryTreeResult{}, fmt.Errorf("run %s does not belong to session %s", input.RunID, input.SessionID)
	}

	touchedFiles, err := s.store.ListTouchedFilesForRun(ctx, input.RunID)
	if err != nil {
		return RepositoryTreeResult{}, err
	}
	result.TouchedFiles = make([]TouchedFileSummary, 0, len(touchedFiles))
	for _, touchedFile := range touchedFiles {
		result.TouchedFiles = append(result.TouchedFiles, TouchedFileSummary{
			RunID:     touchedFile.RunID,
			AgentID:   touchedFile.AgentID,
			FilePath:  touchedFile.FilePath,
			TouchType: touchedFile.TouchType,
		})
	}

	return result, nil
}

func (s *Service) RecordFileTouch(ctx context.Context, filePath string, touchType string) error {
	runContext, ok := runExecutionContextFromContext(ctx)
	if !ok {
		return nil
	}
	normalizedPath := normalizeRepositoryTreePath(filePath)
	if normalizedPath == "" {
		return nil
	}
	agentID := strings.TrimSpace(runContext.AgentID)
	if agentID == "" {
		agentID = strings.TrimSpace(string(runContext.Role))
	}
	if err := s.store.RecordTouchedFile(ctx, sqlite.TouchedFile{
		RunID:      strings.TrimSpace(runContext.RunID),
		AgentID:    agentID,
		FilePath:   normalizedPath,
		TouchType:  strings.TrimSpace(touchType),
		RecordedAt: time.Now().UTC(),
	}); err != nil {
		s.dispatchWorkspaceEnvelope(StreamEnvelope{Type: "error", Payload: map[string]any{
			"code":       "repository_tree_sync_failed",
			"message":    "Relay could not synchronize the repository tree activity indicator. The loaded tree is still available.",
			"session_id": runContext.SessionID,
		}})
		return nil
	}

	s.dispatchWorkspaceEnvelope(StreamEnvelope{Type: "file_touched", Payload: map[string]any{
		"session_id": runContext.SessionID,
		"run_id":     runContext.RunID,
		"agent_id":   agentID,
		"role":       runContext.Role,
		"file_path":  normalizedPath,
		"touch_type": touchType,
		"replay":     false,
	}})
	return nil
}

func normalizeRepositoryTreePath(filePath string) string {
	trimmed := strings.TrimSpace(filePath)
	if trimmed == "" {
		return ""
	}
	cleaned := filepath.ToSlash(filepath.Clean(trimmed))
	if cleaned == "." || strings.HasPrefix(cleaned, "../") {
		return ""
	}
	return cleaned
}
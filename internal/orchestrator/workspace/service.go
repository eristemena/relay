package workspace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/erisristemena/relay/internal/config"
	"github.com/erisristemena/relay/internal/storage/sqlite"
)

type Store interface {
	ListSessions(ctx context.Context) ([]sqlite.Session, error)
	GetSession(ctx context.Context, sessionID string) (sqlite.Session, error)
	CreateSession(ctx context.Context, displayName string) (sqlite.Session, error)
	OpenSession(ctx context.Context, sessionID string) (sqlite.Session, error)
}

type Service struct {
	store Store
	paths config.Paths
}

type SessionSummary struct {
	ID           string    `json:"id"`
	DisplayName  string    `json:"display_name"`
	CreatedAt    time.Time `json:"created_at"`
	LastOpenedAt time.Time `json:"last_opened_at"`
	Status       string    `json:"status"`
	HasActivity  bool      `json:"has_activity"`
}

type UIState struct {
	HistoryState string `json:"history_state"`
	CanvasState  string `json:"canvas_state"`
	SaveState    string `json:"save_state"`
}

type WorkspaceSnapshot struct {
	ActiveSessionID string                 `json:"active_session_id"`
	Sessions        []SessionSummary       `json:"sessions"`
	Preferences     config.SafePreferences `json:"preferences"`
	UIState         UIState                `json:"ui_state"`
	Warnings        []string               `json:"warnings,omitempty"`
}

type CredentialInput struct {
	Provider string `json:"provider"`
	Label    string `json:"label,omitempty"`
	Secret   string `json:"secret"`
}

type PreferencesInput struct {
	PreferredPort     *int
	AppearanceVariant *string
	Credentials       []CredentialInput
	ReplaceCredentials bool
	OpenBrowserOnStart *bool
}

func NewService(store Store, paths config.Paths) *Service {
	return &Service{store: store, paths: paths}
}

func (s *Service) Bootstrap(ctx context.Context, lastSessionID string) (WorkspaceSnapshot, error) {
	cfg, warnings, err := config.Load(s.paths)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}

	sessions, err := s.store.ListSessions(ctx)
	if err != nil {
		return WorkspaceSnapshot{}, fmt.Errorf("load saved sessions: %w", err)
	}

	activeSessionID := strings.TrimSpace(lastSessionID)
	if activeSessionID == "" {
		activeSessionID = strings.TrimSpace(cfg.LastSessionID)
	}

	if activeSessionID != "" {
		if _, err := s.store.GetSession(ctx, activeSessionID); err != nil {
			activeSessionID = ""
			warnings = append(warnings, "The last active session is no longer available, so Relay opened the workspace without restoring it.")
		}
	}

	if activeSessionID == "" && len(sessions) > 0 {
		activeSessionID = sessions[0].ID
	}

	return WorkspaceSnapshot{
		ActiveSessionID: activeSessionID,
		Sessions:        summarizeSessions(sessions),
		Preferences:     cfg.SafePreferences(),
		UIState: UIState{
			HistoryState: "ready",
			CanvasState:  canvasStateForSessions(sessions, activeSessionID),
			SaveState:    "idle",
		},
		Warnings: warnings,
	}, nil
}

func (s *Service) CreateSession(ctx context.Context, displayName string) (WorkspaceSnapshot, error) {
	session, err := s.store.CreateSession(ctx, displayName)
	if err != nil {
		return WorkspaceSnapshot{}, fmt.Errorf("create session: %w", err)
	}

	cfg, warnings, err := config.Load(s.paths)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	cfg.LastSessionID = session.ID
	if err := config.Save(s.paths, cfg); err != nil {
		return WorkspaceSnapshot{}, err
	}

	snapshot, err := s.Bootstrap(ctx, session.ID)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	snapshot.Warnings = append(snapshot.Warnings, warnings...)
	return snapshot, nil
}

func (s *Service) OpenSession(ctx context.Context, sessionID string) (WorkspaceSnapshot, error) {
	session, err := s.store.OpenSession(ctx, sessionID)
	if err != nil {
		return WorkspaceSnapshot{}, fmt.Errorf("open session: %w", err)
	}

	cfg, warnings, err := config.Load(s.paths)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	cfg.LastSessionID = session.ID
	if err := config.Save(s.paths, cfg); err != nil {
		return WorkspaceSnapshot{}, err
	}

	snapshot, err := s.Bootstrap(ctx, session.ID)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	snapshot.Warnings = append(snapshot.Warnings, warnings...)
	return snapshot, nil
}

func (s *Service) SavePreferences(ctx context.Context, input PreferencesInput) (WorkspaceSnapshot, error) {
	cfg, warnings, err := config.Load(s.paths)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}

	if input.PreferredPort != nil {
		if *input.PreferredPort < 1 || *input.PreferredPort > 65535 {
			warnings = append(warnings, "Ignored invalid preferred port and kept the existing saved value.")
		} else {
			cfg.Port = *input.PreferredPort
		}
	}

	if input.AppearanceVariant != nil {
		variant := strings.TrimSpace(*input.AppearanceVariant)
		if variant == "" {
			warnings = append(warnings, "Ignored invalid appearance variant.")
		} else if variant != "midnight" && variant != "graphite" {
			warnings = append(warnings, "Ignored unsupported appearance variant and kept the existing dark-mode setting.")
		} else {
			cfg.AppearanceVariant = variant
		}
	}

	if input.OpenBrowserOnStart != nil {
		cfg.OpenBrowserOnStart = *input.OpenBrowserOnStart
	}

	if input.ReplaceCredentials {
		credentials := make([]config.Credential, 0, len(input.Credentials))
		for _, item := range input.Credentials {
			if strings.TrimSpace(item.Provider) == "" || strings.TrimSpace(item.Secret) == "" {
				warnings = append(warnings, "Ignored an incomplete credential entry.")
				continue
			}
			credentials = append(credentials, config.Credential{
				Provider:  strings.TrimSpace(item.Provider),
				Label:     strings.TrimSpace(item.Label),
				Secret:    item.Secret,
				UpdatedAt: time.Now().UTC(),
			})
		}
		cfg.Credentials = credentials
	}

	if err := config.Save(s.paths, cfg); err != nil {
		return WorkspaceSnapshot{}, err
	}

	snapshot, err := s.Bootstrap(ctx, cfg.LastSessionID)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	snapshot.Warnings = append(snapshot.Warnings, warnings...)
	snapshot.UIState.SaveState = "saved"
	return snapshot, nil
}

func summarizeSessions(sessions []sqlite.Session) []SessionSummary {
	items := make([]SessionSummary, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, SessionSummary{
			ID:           session.ID,
			DisplayName:  session.DisplayName,
			CreatedAt:    session.CreatedAt,
			LastOpenedAt: session.LastOpenedAt,
			Status:       session.Status,
			HasActivity:  session.Snapshot.HasActivity,
		})
	}

	return items
}

func canvasStateForSessions(sessions []sqlite.Session, activeSessionID string) string {
	if len(sessions) == 0 {
		return "empty"
	}

	for _, session := range sessions {
		if session.ID == activeSessionID {
			if session.Snapshot.HasActivity {
				return "ready"
			}
			return "empty"
		}
	}

	return "ready"
}

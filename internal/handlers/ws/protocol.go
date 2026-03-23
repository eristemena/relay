package ws

import "encoding/json"

const (
	TypeWorkspaceBootstrapRequest = "workspace.bootstrap.request"
	TypeWorkspaceBootstrap        = "workspace.bootstrap"
	TypeWorkspaceStatus           = "workspace.status"
	TypeSessionCreate             = "session.create"
	TypeSessionCreated            = "session.created"
	TypeSessionOpen               = "session.open"
	TypeSessionOpened             = "session.opened"
	TypePreferencesSave           = "preferences.save"
	TypePreferencesSaved          = "preferences.saved"
	TypeError                     = "error"
)

type Envelope struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type OutboundEnvelope[T any] struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Payload   T      `json:"payload"`
}

type BootstrapRequestPayload struct {
	LastSessionID string `json:"last_session_id,omitempty"`
}

type SessionCreatePayload struct {
	DisplayName string `json:"display_name,omitempty"`
}

type SessionOpenPayload struct {
	SessionID string `json:"session_id"`
}

type CredentialPayload struct {
	Provider string `json:"provider"`
	Label    string `json:"label,omitempty"`
	Secret   string `json:"secret"`
}

type PreferencesSavePayload struct {
	PreferredPort      *int                `json:"preferred_port,omitempty"`
	AppearanceVariant  *string             `json:"appearance_variant,omitempty"`
	Credentials        []CredentialPayload `json:"credentials,omitempty"`
	OpenBrowserOnStart *bool               `json:"open_browser_on_start,omitempty"`
}

type SessionSummary struct {
	ID           string `json:"id"`
	DisplayName  string `json:"display_name"`
	CreatedAt    string `json:"created_at"`
	LastOpenedAt string `json:"last_opened_at"`
	Status       string `json:"status"`
	HasActivity  bool   `json:"has_activity"`
}

type PreferencesView struct {
	PreferredPort      int    `json:"preferred_port"`
	AppearanceVariant  string `json:"appearance_variant"`
	HasCredentials     bool   `json:"has_credentials"`
	OpenBrowserOnStart bool   `json:"open_browser_on_start"`
}

type UIState struct {
	HistoryState string `json:"history_state"`
	CanvasState  string `json:"canvas_state"`
	SaveState    string `json:"save_state"`
}

type WorkspaceSnapshotPayload struct {
	ActiveSessionID string            `json:"active_session_id"`
	Sessions        []SessionSummary  `json:"sessions"`
	Preferences     PreferencesView   `json:"preferences"`
	UIState         UIState           `json:"ui_state"`
	Warnings        []string          `json:"warnings,omitempty"`
}

type WorkspaceStatusPayload struct {
	Phase   string `json:"phase"`
	Message string `json:"message"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

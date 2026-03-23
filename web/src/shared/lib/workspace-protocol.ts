export type ConnectionMessageType =
  | "workspace.bootstrap.request"
  | "workspace.bootstrap"
  | "workspace.status"
  | "session.create"
  | "session.created"
  | "session.open"
  | "session.opened"
  | "preferences.save"
  | "preferences.saved"
  | "error";

export interface Envelope<TPayload> {
  type: ConnectionMessageType;
  request_id?: string;
  payload: TPayload;
}

export interface WorkspaceBootstrapRequestPayload {
  last_session_id?: string;
}

export interface SessionCreatePayload {
  display_name?: string;
}

export interface SessionOpenPayload {
  session_id: string;
}

export interface CredentialPayload {
  provider: string;
  label?: string;
  secret: string;
}

export interface PreferencesSavePayload {
  preferred_port?: number;
  appearance_variant?: string;
  credentials?: CredentialPayload[];
  open_browser_on_start?: boolean;
}

export interface SessionSummary {
  id: string;
  display_name: string;
  created_at: string;
  last_opened_at: string;
  status: "active" | "idle" | "archived";
  has_activity: boolean;
}

export interface PreferencesView {
  preferred_port: number;
  appearance_variant: string;
  has_credentials: boolean;
  open_browser_on_start: boolean;
}

export interface WorkspaceUIState {
  history_state: "ready" | "loading" | "error";
  canvas_state: "ready" | "empty" | "error";
  save_state: "idle" | "saving" | "saved" | "error";
}

export interface WorkspaceSnapshotPayload {
  active_session_id: string;
  sessions: SessionSummary[];
  preferences: PreferencesView;
  ui_state: WorkspaceUIState;
  warnings?: string[];
}

export interface WorkspaceStatusPayload {
  phase: string;
  message: string;
}

export interface ErrorPayload {
  code: string;
  message: string;
}

export type IncomingEnvelope =
  | Envelope<WorkspaceSnapshotPayload>
  | Envelope<WorkspaceStatusPayload>
  | Envelope<ErrorPayload>;

export function createBootstrapRequest(lastSessionId?: string): Envelope<WorkspaceBootstrapRequestPayload> {
  return {
    type: "workspace.bootstrap.request",
    request_id: crypto.randomUUID(),
    payload: lastSessionId ? { last_session_id: lastSessionId } : {},
  };
}

export function createSessionCreateRequest(displayName?: string): Envelope<SessionCreatePayload> {
  return {
    type: "session.create",
    request_id: crypto.randomUUID(),
    payload: displayName ? { display_name: displayName } : {},
  };
}

export function createSessionOpenRequest(sessionId: string): Envelope<SessionOpenPayload> {
  return {
    type: "session.open",
    request_id: crypto.randomUUID(),
    payload: { session_id: sessionId },
  };
}

export function createPreferencesSaveRequest(payload: PreferencesSavePayload): Envelope<PreferencesSavePayload> {
  return {
    type: "preferences.save",
    request_id: crypto.randomUUID(),
    payload,
  };
}

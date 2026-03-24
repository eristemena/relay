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
  | "agent.run.submit"
  | "agent.run.open"
  | "agent.run.cancel"
  | "agent.run.approval.respond"
  | "approval_request"
  | "state_change"
  | "token"
  | "tool_call"
  | "tool_result"
  | "complete"
  | "agent_spawned"
  | "agent_state_changed"
  | "task_assigned"
  | "handoff_start"
  | "handoff_complete"
  | "agent_error"
  | "run_complete"
  | "run_error"
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
  openrouter_api_key?: string;
  project_root?: string;
  open_browser_on_start?: boolean;
}

export interface AgentRunSubmitPayload {
  session_id: string;
  task: string;
}

export interface AgentRunOpenPayload {
  session_id: string;
  run_id: string;
}

export interface AgentRunCancelPayload {
  session_id: string;
  run_id: string;
}

export interface AgentRunApprovalRespondPayload {
  session_id: string;
  run_id: string;
  tool_call_id: string;
  decision: "approved" | "rejected";
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
  openrouter_configured: boolean;
  project_root: string;
  project_root_configured: boolean;
  project_root_valid: boolean;
  project_root_message?: string;
  agent_models: AgentModelsView;
  open_browser_on_start: boolean;
}

export interface AgentModelsView {
  planner: string;
  coder: string;
  reviewer: string;
  tester: string;
  explainer: string;
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
  active_run_id?: string;
  run_summaries?: AgentRunSummary[];
  credential_status: CredentialStatusView;
  warnings?: string[];
}

export interface AgentRunSummary {
  id: string;
  task_text_preview: string;
  role: "planner" | "coder" | "reviewer" | "tester" | "explainer";
  model: string;
  state:
    | "accepted"
    | "active"
    | "thinking"
    | "tool_running"
    | "approval_required"
    | "completed"
    | "cancelled"
    | "halted"
    | "errored";
  error_code?: string;
  started_at: string;
  completed_at?: string;
  has_tool_activity: boolean;
}

export interface CredentialStatusView {
  configured: boolean;
}

export interface RunEventBase {
  session_id: string;
  run_id: string;
  sequence: number;
  replay: boolean;
  role: AgentRunSummary["role"];
  model: string;
  occurred_at: string;
  agent_id?: string;
}

export interface StateChangePayload extends RunEventBase {
  state: AgentRunSummary["state"];
  message: string;
}

export interface TokenPayload extends RunEventBase {
  text: string;
  first_token_latency_ms?: number;
}

export interface ToolCallPayload extends RunEventBase {
  tool_call_id: string;
  tool_name: string;
  input_preview: Record<string, unknown>;
}

export interface ToolResultPayload extends RunEventBase {
  tool_call_id: string;
  tool_name: string;
  status: string;
  result_preview: Record<string, unknown>;
}

export interface ApprovalRequestPayload {
  session_id: string;
  run_id: string;
  role?: AgentRunSummary["role"];
  model?: string;
  tool_call_id: string;
  tool_name: string;
  input_preview: Record<string, unknown>;
  message: string;
  occurred_at: string;
}

export interface CompletePayload extends RunEventBase {
  finish_reason: string;
}

export interface AgentSpawnedPayload extends RunEventBase {
  agent_id: string;
  label: string;
  spawn_order: number;
}

export interface AgentStateChangedPayload extends RunEventBase {
  agent_id: string;
  state:
    | "queued"
    | "assigned"
    | "thinking"
    | "streaming"
    | "completed"
    | "errored"
    | "cancelled"
    | "blocked";
  message: string;
}

export interface TaskAssignedPayload extends RunEventBase {
  agent_id: string;
  task_text: string;
}

export interface HandoffPayload extends RunEventBase {
  from_agent_id: string;
  to_agent_id: string;
  reason: string;
}

export interface RunCompletePayload extends RunEventBase {
  summary: string;
}

export interface WorkspaceStatusPayload {
  phase: string;
  message: string;
}

export interface ErrorPayload {
  code: string;
  message: string;
  session_id?: string;
  run_id?: string;
  agent_id?: string;
  sequence?: number;
  replay?: boolean;
  role?: AgentRunSummary["role"];
  model?: string;
  terminal?: boolean;
  occurred_at?: string;
}

export type RunEventPayload =
  | StateChangePayload
  | TokenPayload
  | ToolCallPayload
  | ToolResultPayload
  | CompletePayload
  | AgentSpawnedPayload
  | AgentStateChangedPayload
  | TaskAssignedPayload
  | HandoffPayload
  | RunCompletePayload
  | ErrorPayload;

export type IncomingEnvelope =
  | Envelope<WorkspaceSnapshotPayload>
  | Envelope<WorkspaceStatusPayload>
  | Envelope<ErrorPayload>
  | Envelope<ApprovalRequestPayload>
  | Envelope<StateChangePayload>
  | Envelope<TokenPayload>
  | Envelope<ToolCallPayload>
  | Envelope<ToolResultPayload>
  | Envelope<CompletePayload>
  | Envelope<AgentSpawnedPayload>
  | Envelope<AgentStateChangedPayload>
  | Envelope<TaskAssignedPayload>
  | Envelope<HandoffPayload>
  | Envelope<RunCompletePayload>;

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

export function createAgentRunSubmitRequest(
  sessionId: string,
  task: string,
): Envelope<AgentRunSubmitPayload> {
  return {
    type: "agent.run.submit",
    request_id: crypto.randomUUID(),
    payload: { session_id: sessionId, task },
  };
}

export function createAgentRunOpenRequest(
  sessionId: string,
  runId: string,
): Envelope<AgentRunOpenPayload> {
  return {
    type: "agent.run.open",
    request_id: crypto.randomUUID(),
    payload: { session_id: sessionId, run_id: runId },
  };
}

export function createAgentRunCancelRequest(
  sessionId: string,
  runId: string,
): Envelope<AgentRunCancelPayload> {
  return {
    type: "agent.run.cancel",
    request_id: crypto.randomUUID(),
    payload: { session_id: sessionId, run_id: runId },
  };
}

export function createAgentRunApprovalRespondRequest(
  sessionId: string,
  runId: string,
  toolCallId: string,
  decision: AgentRunApprovalRespondPayload["decision"],
): Envelope<AgentRunApprovalRespondPayload> {
  return {
    type: "agent.run.approval.respond",
    request_id: crypto.randomUUID(),
    payload: {
      session_id: sessionId,
      run_id: runId,
      tool_call_id: toolCallId,
      decision,
    },
  };
}

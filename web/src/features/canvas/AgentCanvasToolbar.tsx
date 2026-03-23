"use client";

import { useEffect, useId, useState } from "react";
import {
  agentCanvasRoles,
  agentCanvasStates,
  getRoleLabel,
  type AgentCanvasRole,
  type AgentCanvasState,
  type AgentCanvasNodeModel,
} from "@/features/canvas/canvasModel";

interface AgentCanvasToolbarProps {
  selectedNode: AgentCanvasNodeModel | null;
  onAddNode: (role: AgentCanvasRole | null) => void;
  onApplyState: (state: AgentCanvasState | null) => void;
}

export function AgentCanvasToolbar({
  selectedNode,
  onAddNode,
  onApplyState,
}: AgentCanvasToolbarProps) {
  const addRoleId = useId();
  const stateId = useId();
  const [draftRole, setDraftRole] = useState<AgentCanvasRole | "">("");
  const [draftState, setDraftState] = useState<AgentCanvasState | "">(
    selectedNode?.state ?? "thinking",
  );

  useEffect(() => {
    setDraftState(selectedNode?.state ?? "thinking");
  }, [selectedNode]);

  return (
    <div className="grid gap-4 rounded-[1.5rem] border border-border bg-raised/70 p-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
      <fieldset className="min-w-0 border-0 p-0">
        <legend className="eyebrow">Add local node</legend>
        <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:items-end">
          <label className="flex min-w-0 flex-1 flex-col gap-2 text-sm text-text" htmlFor={addRoleId}>
            Agent role
            <select
              className="rounded-2xl border border-border bg-surface px-4 py-3 text-text"
              id={addRoleId}
              onChange={(event) =>
                setDraftRole(event.target.value as AgentCanvasRole)
              }
              value={draftRole}
            >
              <option value="">Choose a role</option>
              {agentCanvasRoles.map((role) => (
                <option key={role} value={role}>
                  {getRoleLabel(role)}
                </option>
              ))}
            </select>
          </label>
          <button
            className="rounded-full border border-brand-mid bg-brand-mid px-5 py-3 text-sm font-medium text-text"
            onClick={() => onAddNode(draftRole || null)}
            type="button"
          >
            Add node
          </button>
        </div>
        <p className="mt-3 text-sm leading-6 text-text-muted">
          The canvas stays local to this workspace surface. Adding a node updates the graph layout without touching Relay runs or backend state.
        </p>
      </fieldset>

      <fieldset className="min-w-0 border-0 p-0">
        <legend className="eyebrow">Update selected node</legend>
        <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:items-end">
          <label className="flex min-w-0 flex-1 flex-col gap-2 text-sm text-text" htmlFor={stateId}>
            Local node state
            <select
              className="rounded-2xl border border-border bg-surface px-4 py-3 text-text disabled:cursor-not-allowed disabled:opacity-60"
              disabled={!selectedNode}
              id={stateId}
              onChange={(event) =>
                setDraftState(event.target.value as AgentCanvasState)
              }
              value={draftState}
            >
              {agentCanvasStates.map((state) => (
                <option key={state} value={state}>
                  {state.charAt(0).toUpperCase() + state.slice(1)}
                </option>
              ))}
            </select>
          </label>
          <button
            className="rounded-full border border-border bg-surface px-5 py-3 text-sm font-medium text-text disabled:cursor-not-allowed disabled:opacity-60"
            disabled={!selectedNode}
            onClick={() => onApplyState(draftState || null)}
            type="button"
          >
            Apply state
          </button>
        </div>
        <p className="mt-3 text-sm leading-6 text-text-muted">
          {selectedNode
            ? `${selectedNode.label} stays pinned in place while its local visual state changes.`
            : "Select a node on the graph to change its local state without triggering a relayout."}
        </p>
      </fieldset>
    </div>
  );
}
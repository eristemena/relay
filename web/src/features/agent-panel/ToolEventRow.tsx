import type { ToolCallPayload, ToolResultPayload } from "@/shared/lib/workspace-protocol";

interface ToolEventRowProps {
  type: "tool_call" | "tool_result";
  payload: ToolCallPayload | ToolResultPayload;
}

export function ToolEventRow({ type, payload }: ToolEventRowProps) {
  const preview = type === "tool_call" ? (payload as ToolCallPayload).input_preview : (payload as ToolResultPayload).result_preview;

  return (
    <article className="rounded-3xl border border-border bg-raised/70 p-4">
      <div className="flex items-center justify-between gap-3">
        <p className="eyebrow">{type === "tool_call" ? "Tool call" : "Tool result"}</p>
        <span className="text-xs uppercase tracking-[0.18em] text-text-muted">{payload.tool_name}</span>
      </div>
      <pre className="mt-3 overflow-x-auto whitespace-pre-wrap text-sm leading-6 text-text-muted">{JSON.stringify(preview, null, 2)}</pre>
    </article>
  );
}
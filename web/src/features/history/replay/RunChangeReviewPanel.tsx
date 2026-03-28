import type { RunChangeRecordPayload } from "@/shared/lib/workspace-protocol";

interface RunChangeReviewPanelProps {
	changeRecords?: RunChangeRecordPayload[];
	state?: "loading" | "ready" | "empty";
	totalChangeCount?: number;
	selectedTimestampLabel?: string | null;
}

export function RunChangeReviewPanel({
	changeRecords = [],
	state = "empty",
	totalChangeCount,
	selectedTimestampLabel = null,
}: RunChangeReviewPanelProps) {
	const effectiveTotalCount = totalChangeCount ?? changeRecords.length;
	const isCursorFilteredEmpty = state === "ready" && effectiveTotalCount > 0 && changeRecords.length === 0;
	const reviewSummary = state === "ready" && selectedTimestampLabel
		? `Showing ${changeRecords.length} of ${effectiveTotalCount} recorded ${effectiveTotalCount === 1 ? "change" : "changes"} through ${selectedTimestampLabel}.`
		: null;

	return (
		<section aria-labelledby="run-change-review-heading" className="panel-surface rounded-[2rem] p-5 shadow-idle">
			<div>
				<p className="eyebrow">Recorded changes</p>
				<h3 id="run-change-review-heading" className="mt-2 font-display text-2xl text-text">
					Diff review
				</h3>
			</div>
			{state === "loading" ? <p className="mt-6 text-sm text-text-muted">Loading preserved file changes.</p> : null}
			{state === "empty" ? (
				<p className="mt-6 rounded-3xl border border-dashed border-border bg-raised/60 p-5 text-sm leading-6 text-text-muted">
					This saved run does not include recorded file changes.
				</p>
			) : null}
			{reviewSummary ? <p className="mt-6 text-sm leading-6 text-text-muted" role="status">{reviewSummary}</p> : null}
			{isCursorFilteredEmpty ? (
				<p className="mt-6 rounded-3xl border border-dashed border-border bg-raised/60 p-5 text-sm leading-6 text-text-muted">
					No preserved file changes had occurred by this replay position.
				</p>
			) : null}
			{state === "ready" && changeRecords.length > 0 ? (
				<ul className="mt-6 space-y-3">
					{changeRecords.map((record) => (
						<li className="replay-change-card" key={`${record.tool_call_id}:${record.path}`}>
							<div className="flex flex-wrap items-center justify-between gap-3">
								<p className="font-mono text-xs uppercase tracking-[0.18em] text-text-muted">{record.path}</p>
								{record.approval_state ? (
									<p className="rounded-full border border-border px-3 py-1 text-[11px] uppercase tracking-[0.18em] text-text-muted">
										{record.approval_state}
									</p>
								) : null}
							</div>
							<div className="mt-4 grid gap-3 xl:grid-cols-2">
								<div className="replay-change-pane">
									<p className="text-xs uppercase tracking-[0.18em] text-text-muted">Before</p>
									<pre className="mt-3 overflow-x-auto whitespace-pre-wrap break-words font-mono text-xs leading-6 text-text">{record.original_content || "No preserved original content."}</pre>
								</div>
								<div className="replay-change-pane">
									<p className="text-xs uppercase tracking-[0.18em] text-text-muted">After</p>
									<pre className="mt-3 overflow-x-auto whitespace-pre-wrap break-words font-mono text-xs leading-6 text-text">{record.proposed_content || "No preserved proposed content."}</pre>
								</div>
							</div>
						</li>
					))}
				</ul>
			) : null}
		</section>
	);
}
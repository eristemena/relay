"use client";

import { MonacoDiffViewer } from "@/features/approvals/MonacoDiffViewer";
import type { PendingApproval } from "@/shared/lib/workspace-store";

interface ApprovalReviewPanelProps {
	pendingCount?: number;
	selectedApprovalId?: string;
	approval?: PendingApproval | null;
	onApprovalDecision?: (
		toolCallId: string,
		decision: "approved" | "rejected",
	) => void;
}

export function ApprovalReviewPanel({
	pendingCount = 0,
	selectedApprovalId,
	approval,
	onApprovalDecision,
}: ApprovalReviewPanelProps) {
	const hasPendingApprovals = pendingCount > 0;
	const isCommandApproval = approval?.requestKind === "command";
	const isFileWriteApproval = approval?.requestKind === "file_write";
	const approvalOutcome =
		approval?.status === "proposed"
			? "Relay is waiting for your explicit review before it can continue this run."
			: approval?.status
				? `Relay recorded this request as ${approval.status}.`
				: null;

	return (
		<section
			aria-labelledby="approval-review-heading"
			className="panel-surface rounded-[2rem] p-5"
		>
			<div className="flex items-start justify-between gap-4">
				<div>
					<p className="eyebrow">Approval review</p>
					<h2
						id="approval-review-heading"
						className="mt-2 font-display text-2xl text-text"
					>
						Pending write and command requests
					</h2>
				</div>
				<p className="rounded-full border border-border bg-raised px-3 py-1 text-sm text-text-muted">
					{pendingCount} pending
				</p>
			</div>

			{hasPendingApprovals ? (
				<div className="mt-6 space-y-4">
					<p aria-live="polite" className="text-sm leading-6 text-text-muted" role="status">
						Review the exact file diff or command preview before Relay can continue.
					</p>
					{approval ? (
						<div className="space-y-4 rounded-[1.5rem] border border-border bg-raised/70 p-4">
							<div className="flex flex-wrap items-start justify-between gap-3">
								<div>
									<p className="eyebrow">Selected request</p>
									<h3 className="mt-2 font-display text-xl text-text">
										{isCommandApproval
											? "Command review"
											: isFileWriteApproval
												? "File diff review"
												: "Approval review"}
									</h3>
								</div>
								{approval.status ? (
									<p className="rounded-full border border-border bg-base px-3 py-1 text-xs text-text-muted">
										{approval.status}
									</p>
								) : null}
							</div>

							<p className="text-sm leading-6 text-text-muted">{approval.message}</p>
							{approvalOutcome ? (
								<p
									aria-live="polite"
									className="rounded-[1.25rem] border border-border bg-base px-4 py-3 text-sm leading-6 text-text"
									role="status"
								>
									{approvalOutcome}
								</p>
							) : null}

							{approval.repositoryRoot ? (
								<p className="break-all text-xs text-text-muted">
									Repository root: <span className="font-mono text-text">{approval.repositoryRoot}</span>
								</p>
							) : null}

							{approval.diffPreview ? (
								<div className="space-y-3">
									<p className="text-sm text-text">
										Target file: <span className="font-mono">{approval.diffPreview.targetPath}</span>
									</p>
									<MonacoDiffViewer
										originalContent={approval.diffPreview.originalContent}
										proposedContent={approval.diffPreview.proposedContent}
										targetPath={approval.diffPreview.targetPath}
									/>
									<p className="break-all text-xs text-text-muted">
										Base hash: <span className="font-mono text-text">{approval.diffPreview.baseContentHash}</span>
									</p>
								</div>
							) : null}

							{approval.commandPreview ? (
								<div className="space-y-3 rounded-[1.25rem] border border-border bg-base p-3">
									<h4 className="text-sm font-medium text-text">Command preview</h4>
									<p className="break-all text-xs text-text-muted">
										Working directory: <span className="font-mono text-text">{approval.commandPreview.effectiveDir}</span>
									</p>
									<pre className="overflow-x-auto whitespace-pre-wrap break-words rounded-xl border border-border bg-surface p-3 font-mono text-xs text-text">{[approval.commandPreview.command, ...approval.commandPreview.args].join(" ")}</pre>
								</div>
							) : null}

							{onApprovalDecision && approval ? (
								<div className="approval-review-actions flex flex-wrap gap-3">
									<button
										className="rounded-full border border-brand-mid bg-brand-mid px-4 py-2 text-sm font-medium text-text transition duration-200 hover:bg-brand"
										onClick={() => onApprovalDecision(approval.toolCallId, "approved")}
										type="button"
									>
										Approve request
									</button>
									<button
										className="rounded-full border border-border bg-raised px-4 py-2 text-sm font-medium text-text transition duration-200 hover:border-error"
										onClick={() => onApprovalDecision(approval.toolCallId, "rejected")}
										type="button"
									>
										Reject request
									</button>
								</div>
							) : null}
						</div>
					) : (
						<p aria-live="polite" className="rounded-3xl border border-dashed border-border bg-raised/60 p-5 text-sm leading-6 text-text-muted" role="status">
							Choose a pending approval to inspect its exact diff or command preview.
						</p>
					)}
				</div>
			) : (
				<p aria-live="polite" className="mt-6 rounded-3xl border border-dashed border-border bg-raised/60 p-5 text-sm leading-6 text-text-muted" role="status">
					No approvals are waiting for review. When an agent proposes a file write or command,
					 Relay will keep it here until you approve or reject it.
				</p>
			)}

			{selectedApprovalId ? (
				<p className="mt-4 text-sm text-text-muted">
					Selected approval: <span className="font-mono text-text">{selectedApprovalId}</span>
				</p>
			) : null}
		</section>
	);
}
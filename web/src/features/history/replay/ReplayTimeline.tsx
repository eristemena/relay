import { useEffect, useState } from "react";

interface ReplayTimelineProps {
	cursorMs?: number;
	durationMs?: number;
	onSeek?: (cursorMs: number) => void;
}

export function ReplayTimeline({ cursorMs = 0, durationMs = 0, onSeek }: ReplayTimelineProps) {
	const boundedDuration = Math.max(durationMs, 0);
	const boundedCursor = Math.min(cursorMs, boundedDuration);
	const [draftCursorMs, setDraftCursorMs] = useState(boundedCursor);
	const [isDragging, setIsDragging] = useState(false);

	useEffect(() => {
		if (isDragging) {
			return;
		}
		setDraftCursorMs(boundedCursor);
	}, [boundedCursor, isDragging]);

	function commitSeek(nextCursorMs: number) {
		const clampedCursor = Math.min(Math.max(nextCursorMs, 0), boundedDuration);
		setDraftCursorMs(clampedCursor);
		onSeek?.(clampedCursor);
	}

	return (
		<section aria-labelledby="replay-timeline-heading" className="panel-surface rounded-[2rem] p-5 shadow-idle">
			<div>
				<p className="eyebrow">Replay timeline</p>
				<h3 id="replay-timeline-heading" className="mt-2 font-display text-2xl text-text">
					Scrubber
				</h3>
			</div>
			<input
				aria-label="Replay position"
				aria-valuetext={`${draftCursorMs} milliseconds`}
				className="replay-timeline-input mt-6 w-full accent-[var(--color-brand-mid)]"
				disabled={!onSeek}
				max={boundedDuration}
				min={0}
				onChange={(event) => {
					const nextCursorMs = Number(event.currentTarget.value);
					setDraftCursorMs(nextCursorMs);
					if (!isDragging) {
						commitSeek(nextCursorMs);
					}
				}}
				onPointerDown={() => setIsDragging(true)}
				onPointerUp={(event) => {
					setIsDragging(false);
					commitSeek(Number(event.currentTarget.value));
				}}
				onKeyUp={(event) => {
					commitSeek(Number(event.currentTarget.value));
				}}
				type="range"
				value={draftCursorMs}
			/>
			<div className="mt-3 flex items-center justify-between text-xs uppercase tracking-[0.18em] text-text-muted">
				<span>{draftCursorMs} ms</span>
				<span>{boundedDuration} ms</span>
			</div>
		</section>
	);
}
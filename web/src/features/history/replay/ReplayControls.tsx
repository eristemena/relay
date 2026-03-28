interface ReplayControlsProps {
	status?: "preparing" | "playing" | "paused" | "seeking" | "completed" | "error";
	speed?: 0.5 | 1 | 2 | 5;
	onPlay?: () => void;
	onPause?: () => void;
	onReset?: () => void;
	onSpeedChange?: (speed: 0.5 | 1 | 2 | 5) => void;
	onExport?: () => void;
	exportStatus?: "started" | "completed" | "error";
}

const speedOptions: Array<0.5 | 1 | 2 | 5> = [0.5, 1, 2, 5];

export function ReplayControls({
	status = "paused",
	speed = 1,
	onPlay,
	onPause,
	onReset,
	onSpeedChange,
	onExport,
	exportStatus,
}: ReplayControlsProps) {
	const canPlay = status === "paused" || status === "completed";
	const canPause = status === "playing" && Boolean(onPause);
	const canReset = Boolean(onReset);
	const canExport = Boolean(onExport);
	const canAdjustSpeed = Boolean(onSpeedChange);

	return (
		<section aria-labelledby="replay-controls-heading" className="panel-surface rounded-[2rem] p-5 shadow-idle">
			<div className="flex flex-wrap items-start justify-between gap-4">
				<div>
					<p className="eyebrow">Replay controls</p>
					<h3 id="replay-controls-heading" className="mt-2 font-display text-2xl text-text">
						Playback
					</h3>
				</div>
				<p className="rounded-full border border-border px-3 py-1 text-xs uppercase tracking-[0.18em] text-text-muted">
					{status}
				</p>
			</div>
			<div className="mt-5 flex flex-wrap gap-3">
				<button className="replay-control-button" disabled={!canPlay || !onPlay} onClick={onPlay} type="button">
					Play
				</button>
				<button className="replay-control-button" disabled={!canPause} onClick={onPause} type="button">
					Pause
				</button>
				<button className="replay-control-button" disabled={!canReset} onClick={onReset} type="button">
					Reset
				</button>
				<button className="replay-control-button" data-variant="accent" disabled={!canExport} onClick={onExport} type="button">
					Export Report
				</button>
			</div>
			<fieldset className="mt-5">
				<legend className="text-sm text-text-muted">Playback speed</legend>
				<div className="mt-3 flex flex-wrap gap-2">
					{speedOptions.map((option) => (
						<button
							aria-pressed={option === speed}
							className="replay-speed-button"
							disabled={!canAdjustSpeed}
							key={option}
							onClick={() => onSpeedChange?.(option)}
							type="button"
						>
							{option}x
						</button>
					))}
				</div>
			</fieldset>
			<p className="mt-5 text-sm text-text-muted">
				{exportStatus === "error"
					? "Relay could not export this historical run yet."
					: exportStatus === "completed"
						? "Relay saved the historical report to your local exports folder."
						: "Historical playback stays read-only. Export writes only when you explicitly request it."}
			</p>
		</section>
	);
}
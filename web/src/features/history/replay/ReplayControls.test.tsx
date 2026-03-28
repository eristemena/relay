import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ReplayControls } from "@/features/history/replay/ReplayControls";

describe("ReplayControls", () => {
	it("dispatches playback actions and speed changes", () => {
		const onPlay = vi.fn();
		const onPause = vi.fn();
		const onReset = vi.fn();
		const onSpeedChange = vi.fn();
		const onExport = vi.fn();

		render(
			<ReplayControls
				exportStatus="started"
				onExport={onExport}
				onPause={onPause}
				onPlay={onPlay}
				onReset={onReset}
				onSpeedChange={onSpeedChange}
				status="paused"
				speed={1}
			/>,
		);

		fireEvent.click(screen.getByRole("button", { name: /play/i }));
		fireEvent.click(screen.getByRole("button", { name: /reset/i }));
		fireEvent.click(screen.getByRole("button", { name: /2x/i }));
		fireEvent.click(screen.getByRole("button", { name: /export report/i }));

		expect(onPlay).toHaveBeenCalledTimes(1);
		expect(onReset).toHaveBeenCalledTimes(1);
		expect(onSpeedChange).toHaveBeenCalledWith(2);
		expect(onExport).toHaveBeenCalledTimes(1);
		expect(screen.getByRole("button", { name: /pause/i })).toBeDisabled();
		expect(
			screen.getByText(/historical playback stays read-only/i),
		).toBeInTheDocument();
	});

	it("shows the export error copy and enables pause while playing", () => {
		const onPause = vi.fn();

		render(
			<ReplayControls
				exportStatus="error"
				onPause={onPause}
				status="playing"
				speed={5}
			/>,
		);

		const playButton = screen.getByRole("button", { name: /play/i });
		const pauseButton = screen.getByRole("button", { name: /pause/i });

		expect(playButton).toBeDisabled();
		expect(pauseButton).toBeEnabled();
		fireEvent.click(pauseButton);
		expect(onPause).toHaveBeenCalledTimes(1);
		expect(screen.getByRole("button", { name: /^5x$/i })).toHaveAttribute(
			"aria-pressed",
			"true",
		);
		expect(
			screen.getByText(/relay could not export this historical run yet/i),
		).toBeInTheDocument();
	});
});
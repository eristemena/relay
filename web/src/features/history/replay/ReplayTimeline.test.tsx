import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ReplayTimeline } from "@/features/history/replay/ReplayTimeline";

describe("ReplayTimeline", () => {
	it("renders cursor and duration and dispatches seek events", () => {
		const onSeek = vi.fn();

		render(<ReplayTimeline cursorMs={1200} durationMs={5000} onSeek={onSeek} />);

		const slider = screen.getByLabelText(/replay position/i);
		expect(slider).toHaveAttribute("max", "5000");
		expect(slider).toHaveValue("1200");
		expect(screen.getByText("1200 ms")).toBeInTheDocument();
		expect(screen.getByText("5000 ms")).toBeInTheDocument();

		fireEvent.change(slider, { target: { value: "2400" } });
		expect(onSeek).toHaveBeenCalledWith(2400);
	});

	it("keeps dragging local until the pointer is released", () => {
		const onSeek = vi.fn();

		render(<ReplayTimeline cursorMs={1200} durationMs={5000} onSeek={onSeek} />);

		const slider = screen.getByLabelText(/replay position/i);
		fireEvent.pointerDown(slider);
		fireEvent.change(slider, { target: { value: "2400" } });

		expect(onSeek).not.toHaveBeenCalled();
		expect(screen.getByText("2400 ms")).toBeInTheDocument();

		fireEvent.pointerUp(slider, { target: { value: "2400" } });
		expect(onSeek).toHaveBeenCalledWith(2400);
	});

	it("clamps cursor to the available duration", () => {
		render(<ReplayTimeline cursorMs={9000} durationMs={3000} onSeek={() => undefined} />);

		expect(screen.getByLabelText(/replay position/i)).toHaveValue("3000");
		expect(screen.getAllByText("3000 ms")).toHaveLength(2);
	});
});
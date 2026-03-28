import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RunChangeReviewPanel } from "@/features/history/replay/RunChangeReviewPanel";

describe("RunChangeReviewPanel", () => {
	it("renders loading and empty states", () => {
		const { rerender } = render(<RunChangeReviewPanel state="loading" />);

		expect(
			screen.getByText(/loading preserved file changes/i),
		).toBeInTheDocument();

		rerender(<RunChangeReviewPanel state="empty" />);
		expect(
			screen.getByText(/does not include recorded file changes/i),
		).toBeInTheDocument();
	});

	it("renders the filtered replay summary and hides future changes", () => {
		render(
			<RunChangeReviewPanel
				changeRecords={[
					{
						tool_call_id: "call_1",
						path: "README.md",
						original_content: "before\n",
						proposed_content: "after\n",
						approval_state: "applied",
					},
				]}
				selectedTimestampLabel="3/24/2026, 12:00:02 PM"
				state="ready"
				totalChangeCount={2}
			/>,
		);

		expect(
			screen.getByText(/showing 1 of 2 recorded changes through/i),
		).toBeInTheDocument();
		expect(screen.getByText(/readme\.md/i)).toBeInTheDocument();
		expect(screen.getByText(/^applied$/i)).toBeInTheDocument();
	});

	it("shows a cursor-specific empty state when no change existed yet", () => {
		render(
			<RunChangeReviewPanel
				changeRecords={[]}
				selectedTimestampLabel="3/24/2026, 12:00:00 PM"
				state="ready"
				totalChangeCount={2}
			/>,
		);

		expect(
			screen.getByText(/no preserved file changes had occurred by this replay position/i),
		).toBeInTheDocument();
	});
});
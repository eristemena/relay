import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { ApprovalReviewPanel } from "@/features/approvals/ApprovalReviewPanel";

vi.mock("@/features/approvals/MonacoDiffViewer", () => ({
	MonacoDiffViewer: ({ targetPath }: { targetPath: string }) => (
		<div data-testid="monaco-diff-viewer">Diff viewer for {targetPath}</div>
	),
}));

describe("ApprovalReviewPanel", () => {
	it("renders an empty review state when no approvals are pending", () => {
		render(<ApprovalReviewPanel />);

		expect(
			screen.getByRole("heading", { name: /pending write and command requests/i }),
		).toBeInTheDocument();
		expect(screen.getByText(/0 pending/i)).toBeInTheDocument();
		expect(
			screen.getByText(/no approvals are waiting for review/i),
		).toBeInTheDocument();
	});

	it("describes the active approval surface when pending work exists", () => {
		render(
			<ApprovalReviewPanel pendingCount={2} selectedApprovalId="approval_123" />,
		);

		expect(screen.getByText(/2 pending/i)).toBeInTheDocument();
		expect(
			screen.getByText(/review the exact file diff or command preview/i),
		).toBeInTheDocument();
		expect(screen.getByText(/approval_123/i)).toBeInTheDocument();
	});

	it("renders a file diff preview and forwards review decisions", async () => {
		const user = userEvent.setup();
		const decisions: Array<{ toolCallId: string; decision: "approved" | "rejected" }> = [];

		render(
			<ApprovalReviewPanel
				approval={{
					sessionId: "session_alpha",
					runId: "run_1",
					toolCallId: "call_1",
					toolName: "write_file",
					requestKind: "file_write",
					status: "proposed",
					repositoryRoot: "/tmp/project",
					inputPreview: { path: "README.md" },
					diffPreview: {
						targetPath: "README.md",
						originalContent: "before\n",
						proposedContent: "after\n",
						baseContentHash: "sha256:abc",
					},
					message:
						"Relay needs approval before it can write files inside the configured project root.",
					occurredAt: "2026-03-26T12:00:00Z",
				}}
				pendingCount={1}
				selectedApprovalId="call_1"
				onApprovalDecision={(toolCallId, decision) =>
					decisions.push({ toolCallId, decision })
				}
			/>,
		);

		expect(screen.getByText(/file diff review/i)).toBeInTheDocument();
		expect(screen.getByText(/target file:/i)).toBeInTheDocument();
		expect(screen.getByTestId("monaco-diff-viewer")).toHaveTextContent(
			/readme\.md/i,
		);
		expect(screen.getByText(/sha256:abc/i)).toBeInTheDocument();
		expect(
			screen.getByText(/waiting for your explicit review/i),
		).toBeInTheDocument();

		await user.click(screen.getByRole("button", { name: /approve request/i }));
		await user.click(screen.getByRole("button", { name: /reject request/i }));

		expect(decisions).toEqual([
			{ toolCallId: "call_1", decision: "approved" },
			{ toolCallId: "call_1", decision: "rejected" },
		]);
	});

	it("renders a command preview when the selected approval is for run_command", () => {
		render(
			<ApprovalReviewPanel
				approval={{
					sessionId: "session_alpha",
					runId: "run_1",
					toolCallId: "call_2",
					toolName: "run_command",
					requestKind: "command",
					repositoryRoot: "/tmp/project",
					inputPreview: { command: "go", args: ["test", "./..."] },
					commandPreview: {
						command: "go",
						args: ["test", "./..."],
						effectiveDir: "/tmp/project",
					},
					message:
						"Relay needs approval before it can run a shell command from the configured project root.",
					occurredAt: "2026-03-26T12:00:00Z",
				}}
				pendingCount={1}
				selectedApprovalId="call_2"
			/>,
		);

		expect(screen.getByText(/command review/i)).toBeInTheDocument();
		expect(screen.getByText(/working directory:/i)).toBeInTheDocument();
		expect(screen.getByText("go test ./...")).toBeInTheDocument();
	});
});
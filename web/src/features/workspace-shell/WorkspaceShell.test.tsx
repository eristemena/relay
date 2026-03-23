import { act, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { WorkspaceShell } from "@/features/workspace-shell/WorkspaceShell";
import { buildWorkspaceSnapshot, primeWorkspaceStore } from "@/shared/lib/test-helpers";
import { resetWorkspaceStore, workspaceStore } from "@/shared/lib/workspace-store";

const socketActions = {
  createSession: vi.fn(),
  openSession: vi.fn(),
  savePreferences: vi.fn(),
};

vi.mock("@/shared/lib/useWorkspaceSocket", () => ({
  useWorkspaceSocket: () => socketActions,
}));

describe("WorkspaceShell", () => {
  beforeEach(() => {
    resetWorkspaceStore();
    vi.clearAllMocks();
  });

  it("renders the initial loading state", () => {
    render(<WorkspaceShell />);

    expect(screen.getByText(/connecting to the relay workspace/i)).toBeInTheDocument();
  });

  it("renders the ready workspace state", () => {
    primeWorkspaceStore(buildWorkspaceSnapshot());
    render(<WorkspaceShell />);

    expect(screen.getByRole("heading", { name: /local ai session control/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /open session: inspect relay startup/i })).toBeInTheDocument();
  });

  it("renders a recoverable error state", () => {
    primeWorkspaceStore(buildWorkspaceSnapshot());
    render(<WorkspaceShell />);

    act(() => {
      workspaceStore.handleEnvelope({
        type: "error",
        payload: {
          code: "session_not_found",
          message: "That session is no longer available. Choose another session or start a new one.",
        },
      } as never);
    });

    expect(screen.getByRole("alert")).toHaveTextContent(/that session is no longer available/i);
  });
});

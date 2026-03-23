import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SessionSidebar } from "@/features/history/SessionSidebar";
import { buildWorkspaceSnapshot, renderWithWorkspace } from "@/shared/lib/test-helpers";

describe("SessionSidebar", () => {
  it("renders the explicit empty-history state", () => {
    renderWithWorkspace(
      <SessionSidebar activeSessionId="" onCreate={vi.fn()} onOpen={vi.fn()} sessions={[]} />,
      buildWorkspaceSnapshot({ active_session_id: "", sessions: [] }),
    );

    expect(screen.getByText(/no saved sessions yet/i)).toBeInTheDocument();
  });

  it("opens the selected session", () => {
    const onOpen = vi.fn();
    renderWithWorkspace(
      <SessionSidebar
        activeSessionId="session_alpha"
        onCreate={vi.fn()}
        onOpen={onOpen}
        sessions={[
          {
            id: "session_alpha",
            display_name: "Inspect relay startup",
            created_at: "2026-03-23T12:00:00Z",
            last_opened_at: "2026-03-23T12:15:00Z",
            status: "active",
            has_activity: false,
          },
        ]}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /open session: inspect relay startup/i }));
    expect(onOpen).toHaveBeenCalledWith("session_alpha");
  });
});

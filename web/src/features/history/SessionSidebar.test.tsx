import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SessionSidebar } from "@/features/history/SessionSidebar";
import { buildWorkspaceSnapshot, renderWithWorkspace } from "@/shared/lib/test-helpers";

describe("SessionSidebar", () => {
  it("renders the empty project-context state", () => {
    renderWithWorkspace(
      <SessionSidebar
        activeProjectRoot=""
        knownProjects={[]}
        onOpenPreferences={vi.fn()}
        onSwitch={vi.fn()}
      />,
      buildWorkspaceSnapshot({ active_session_id: "", sessions: [] }),
    );

    expect(
      screen.getByText(/relay has not connected a project root yet/i),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /open local settings/i }),
    ).toBeInTheDocument();
  });

  it("opens local settings instead of manual session controls", () => {
    const onOpenPreferences = vi.fn();
    renderWithWorkspace(
      <SessionSidebar
        activeProjectRoot="/tmp/relay"
        knownProjects={[
          {
            project_root: "/tmp/relay",
            label: "relay",
            is_active: true,
            is_available: true,
            last_opened_at: "2026-03-23T12:15:00Z",
          },
        ]}
        onOpenPreferences={onOpenPreferences}
        onSwitch={vi.fn()}
      />,
    );

    fireEvent.click(
      screen.getByRole("button", { name: /open local settings/i }),
    );
    expect(onOpenPreferences).toHaveBeenCalledTimes(1);
    expect(screen.queryByText(/start new session/i)).not.toBeInTheDocument();
  });

  it("switches projects from the project context panel", () => {
    const onSwitch = vi.fn();

    renderWithWorkspace(
      <SessionSidebar
        activeProjectRoot="/tmp/relay"
        knownProjects={[
          {
            project_root: "/tmp/relay",
            label: "relay",
            is_active: true,
            is_available: true,
            last_opened_at: "2026-03-23T12:15:00Z",
          },
          {
            project_root: "/tmp/another",
            label: "another",
            is_active: false,
            is_available: true,
            last_opened_at: "2026-03-23T12:20:00Z",
          },
        ]}
        onOpenPreferences={vi.fn()}
        onSwitch={onSwitch}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /another/i }));

    expect(onSwitch).toHaveBeenCalledWith("/tmp/another");
  });
});

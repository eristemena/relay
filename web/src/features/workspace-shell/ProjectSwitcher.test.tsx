import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ProjectSwitcher } from "@/features/workspace-shell/ProjectSwitcher";

describe("ProjectSwitcher", () => {
  it("renders the single-project empty state", () => {
    render(
      <ProjectSwitcher
        activeProjectRoot="/tmp/relay"
        knownProjects={[
          {
            project_root: "/tmp/relay",
            label: "relay",
            is_active: true,
            is_available: true,
            last_opened_at: "2026-04-05T12:34:56Z",
          },
        ]}
        onSwitch={() => {}}
      />,
    );

    expect(screen.getByText(/relay only knows the current project so far/i)).toBeInTheDocument();
    expect(screen.getByText("/tmp/relay")).toBeInTheDocument();
  });

  it("renders guidance when no active project root is selected yet", () => {
    render(
      <ProjectSwitcher
        activeProjectRoot=""
        knownProjects={[]}
        onSwitch={() => undefined}
      />,
    );

    expect(screen.getByText(/no active project selected yet/i)).toBeInTheDocument();
    expect(
      screen.getByText(/open local settings to choose the first project root for this workspace/i),
    ).toBeInTheDocument();
  });

  it("fires the switch callback for an alternate project", () => {
    const onSwitch = vi.fn();

    render(
      <ProjectSwitcher
        activeProjectRoot="/tmp/relay"
        knownProjects={[
          {
            project_root: "/tmp/relay",
            label: "relay",
            is_active: true,
            is_available: true,
            last_opened_at: "2026-04-05T12:34:56Z",
          },
          {
            project_root: "/tmp/another",
            label: "another",
            is_active: false,
            is_available: true,
            last_opened_at: "2026-04-05T12:40:00Z",
          },
        ]}
        onSwitch={onSwitch}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /another/i }));

    expect(onSwitch).toHaveBeenCalledTimes(1);
    expect(onSwitch).toHaveBeenCalledWith("/tmp/another");
  });

  it("renders blocked and unavailable projects as disabled with context", () => {
    render(
      <ProjectSwitcher
        activeProjectRoot="/tmp/relay"
        knownProjects={[
          {
            project_root: "/tmp/relay",
            label: "relay",
            is_active: true,
            is_available: true,
            last_opened_at: "2026-04-05T12:34:56Z",
          },
          {
            project_root: "/tmp/busy",
            label: "busy",
            is_active: false,
            is_available: true,
            blocked_reason:
              "Finish or stop the active run before switching projects.",
            last_opened_at: "2026-04-05T12:40:00Z",
          },
          {
            project_root: "/tmp/missing",
            label: "missing",
            is_active: false,
            is_available: false,
            last_opened_at: "2026-04-05T12:41:00Z",
          },
        ]}
        onSwitch={() => undefined}
      />,
    );

    expect(screen.getByRole("button", { name: /busy/i })).toBeDisabled();
    expect(
      screen.getByText(/finish or stop the active run before switching projects/i),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /missing/i })).toBeDisabled();
    expect(
      screen.getByText(
        /relay cannot switch to this project until the path becomes available again/i,
      ),
    ).toBeInTheDocument();
  });
});
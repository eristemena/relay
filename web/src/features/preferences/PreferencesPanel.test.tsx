import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PreferencesPanel } from "@/features/preferences/PreferencesPanel";

describe("PreferencesPanel", () => {
  it("submits the edited preference values", () => {
    const onSave = vi.fn();
    render(
      <PreferencesPanel
        onSave={onSave}
        preferences={{
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: false,
          openrouter_configured: false,
          project_root: "",
          project_root_configured: false,
          project_root_valid: false,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-flash-1.5",
          },
          open_browser_on_start: true,
        }}
        saveState="idle"
      />,
    );

    fireEvent.change(screen.getByLabelText(/preferred port/i), {
      target: { value: "4848" },
    });
    fireEvent.change(screen.getByLabelText(/appearance variant/i), {
      target: { value: "graphite" },
    });
    fireEvent.change(screen.getByLabelText(/project root/i), {
      target: { value: "/tmp/project" },
    });
    fireEvent.change(screen.getByLabelText(/openrouter api key/i), {
      target: { value: "secret-value" },
    });
    fireEvent.click(screen.getByRole("button", { name: /save preferences/i }));

    expect(onSave).toHaveBeenCalledWith({
      preferred_port: 4848,
      appearance_variant: "graphite",
      open_browser_on_start: true,
      openrouter_api_key: "secret-value",
      project_root: "/tmp/project",
    });
  });

  it("hydrates project root from updated preferences after mount", () => {
    const { rerender } = render(
      <PreferencesPanel
        onSave={() => undefined}
        preferences={{
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: false,
          openrouter_configured: false,
          project_root: "",
          project_root_configured: false,
          project_root_valid: false,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-flash-1.5",
          },
          open_browser_on_start: true,
        }}
        saveState="idle"
      />,
    );

    rerender(
      <PreferencesPanel
        onSave={() => undefined}
        preferences={{
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: false,
          openrouter_configured: false,
          project_root: "/tmp/project",
          project_root_configured: true,
          project_root_valid: true,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-flash-1.5",
          },
          open_browser_on_start: true,
        }}
        saveState="idle"
      />,
    );

    expect(screen.getByLabelText(/project root/i)).toHaveValue("/tmp/project");
  });
});
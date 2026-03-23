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
          open_browser_on_start: true,
        }}
        saveState="idle"
      />,
    );

    fireEvent.change(screen.getByLabelText(/preferred port/i), { target: { value: "4848" } });
    fireEvent.change(screen.getByLabelText(/appearance variant/i), { target: { value: "graphite" } });
    fireEvent.change(screen.getByLabelText(/credential provider/i), { target: { value: "openai" } });
    fireEvent.change(screen.getByLabelText(/credential label/i), { target: { value: "Personal" } });
    fireEvent.change(screen.getByLabelText(/api credential secret/i), { target: { value: "secret-value" } });
    fireEvent.click(screen.getByRole("button", { name: /save preferences/i }));

    expect(onSave).toHaveBeenCalledWith({
      preferred_port: 4848,
      appearance_variant: "graphite",
      open_browser_on_start: true,
      credentials: [
        {
          provider: "openai",
          label: "Personal",
          secret: "secret-value",
        },
      ],
    });
  });
});
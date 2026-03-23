import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StateBadge } from "./StateBadge";

describe("StateBadge", () => {
  it("renders the approval-required label and glow state", () => {
    render(<StateBadge state="approval_required" />);

    const badge = screen.getByText("Approval required");
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain("state-glow-thinking");
  });

  it("renders terminal states with their labels", () => {
    const { rerender } = render(<StateBadge state="completed" />);

    expect(screen.getByText("Completed").className).toContain("state-glow-complete");

    rerender(<StateBadge state="errored" />);

    expect(screen.getByText("Errored").className).toContain("state-glow-error");
  });
});
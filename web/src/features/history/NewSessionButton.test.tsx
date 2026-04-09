import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { NewSessionButton } from "@/features/history/NewSessionButton";

describe("NewSessionButton", () => {
  it("fires the click callback with project-context copy", () => {
    const onClick = vi.fn();
    render(<NewSessionButton onClick={onClick} />);

    fireEvent.click(
      screen.getByRole("button", { name: /open local settings/i }),
    );
    expect(onClick).toHaveBeenCalledTimes(1);
  });
});

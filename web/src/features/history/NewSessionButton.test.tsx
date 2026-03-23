import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { NewSessionButton } from "@/features/history/NewSessionButton";

describe("NewSessionButton", () => {
  it("fires the create callback", () => {
    const onCreate = vi.fn();
    render(<NewSessionButton onCreate={onCreate} />);

    fireEvent.click(screen.getByRole("button", { name: /start new session/i }));
    expect(onCreate).toHaveBeenCalledTimes(1);
  });
});

"use client";

interface NewSessionButtonProps {
  label?: string;
  onClick: () => void;
}

export function NewSessionButton({
  label = "Open local settings",
  onClick,
}: NewSessionButtonProps) {
  return (
    <button
      className="rounded-full border border-brand-mid bg-brand-mid px-4 py-2 font-medium text-text transition duration-300 ease-relay hover:bg-brand"
      onClick={onClick}
      type="button"
    >
      {label}
    </button>
  );
}

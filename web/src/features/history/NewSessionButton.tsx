"use client";

interface NewSessionButtonProps {
  onCreate: () => void;
}

export function NewSessionButton({ onCreate }: NewSessionButtonProps) {
  return (
    <button
      className="rounded-full border border-brand-mid bg-brand-mid px-4 py-2 font-medium text-text transition duration-300 ease-relay hover:bg-brand"
      onClick={onCreate}
      type="button"
    >
      Start new session
    </button>
  );
}

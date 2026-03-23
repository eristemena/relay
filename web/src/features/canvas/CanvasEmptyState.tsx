"use client";

export function CanvasEmptyState({ sessionLabel }: { sessionLabel: string }) {
  return (
    <div className="panel-surface relative overflow-hidden rounded-[2rem] p-6 shadow-idle">
      <div className="max-w-xl">
        <p className="eyebrow">Fresh workspace</p>
        <h2 className="mt-3 font-display text-3xl tracking-tight text-text">{sessionLabel} is ready for its first task.</h2>
        <p className="mt-4 text-base leading-7 text-text-muted">
          This canvas stays intentionally quiet until the session gains activity. Use the sidebar to resume another session or save preferences for the next run.
        </p>
      </div>
    </div>
  );
}

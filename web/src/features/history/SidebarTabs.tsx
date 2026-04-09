"use client";

import type { KeyboardEvent } from "react";

interface SidebarTabsProps {
  activeTab: "replay" | "repository_tree";
  disabled?: boolean;
  onChange: (tab: "replay" | "repository_tree") => void;
}

export function SidebarTabs({
  activeTab,
  disabled = false,
  onChange,
}: SidebarTabsProps) {
  const tabs: Array<{
    value: "replay" | "repository_tree";
    label: string;
    caption: string;
    id: string;
    panelId: string;
  }> = [
    {
      value: "replay",
      label: "Historical replay",
      caption: "Timeline and controls",
      id: "replay-tab",
      panelId: "replay-tabpanel",
    },
    {
      value: "repository_tree",
      label: "Repository tree",
      caption: "Live file map",
      id: "repository-tree-tab",
      panelId: "repository-tree-tabpanel",
    },
  ];

  function handleKeyDown(event: KeyboardEvent<HTMLButtonElement>) {
    if (disabled) {
      return;
    }

    if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) {
      return;
    }

    event.preventDefault();
    const currentIndex = tabs.findIndex((tab) => tab.value === activeTab);
    if (currentIndex === -1) {
      return;
    }

    const nextIndex =
      event.key === "Home"
        ? 0
        : event.key === "End"
          ? tabs.length - 1
          : event.key === "ArrowRight"
            ? (currentIndex + 1) % tabs.length
            : (currentIndex - 1 + tabs.length) % tabs.length;

    onChange(tabs[nextIndex].value);
  }

  return (
    <div
      aria-label="Run detail tabs"
      className="repository-tabs panel-surface flex gap-2 rounded-[999px] p-2"
      role="tablist"
    >
      {tabs.map((tab) => (
        <button
          aria-controls={tab.panelId}
          aria-disabled={disabled}
          aria-selected={activeTab === tab.value}
          className="repository-tab flex-1 rounded-full border border-border px-4 py-2 text-sm font-medium text-text transition-colors duration-200 data-[active=true]:border-brand-mid data-[active=true]:bg-raised disabled:cursor-not-allowed disabled:opacity-60"
          data-active={activeTab === tab.value}
          disabled={disabled}
          id={tab.id}
          key={tab.value}
          onClick={() => {
            if (!disabled) {
              onChange(tab.value);
            }
          }}
          onKeyDown={handleKeyDown}
          role="tab"
          tabIndex={disabled ? -1 : activeTab === tab.value ? 0 : -1}
          type="button"
        >
          <span className="block">{tab.label}</span>
          <span className="repository-tab-caption block text-xs font-normal text-text-muted">
            {tab.caption}
          </span>
        </button>
      ))}
    </div>
  );
}
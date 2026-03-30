import { describe, expect, it } from "vitest";
import { buildRepositoryTreeView } from "@/features/history/treeModel";

describe("buildRepositoryTreeView", () => {
  it("keeps deeper descendants hidden until their parent directories are expanded", () => {
    const initialView = buildRepositoryTreeView({
      paths: [
        "README.md",
        "docs",
        "docs/guides",
        "docs/guides/setup.md",
        "docs/guides/deep",
        "docs/guides/deep/notes.md",
      ],
      touchedFiles: [],
      selectedAgentId: null,
      expandedPaths: [],
    });

    expect(initialView.entries.map((entry) => entry.path)).toEqual([
      "README.md",
      "docs",
      "docs/guides",
    ]);

    const expandedView = buildRepositoryTreeView({
      paths: [
        "README.md",
        "docs",
        "docs/guides",
        "docs/guides/setup.md",
        "docs/guides/deep",
        "docs/guides/deep/notes.md",
      ],
      touchedFiles: [],
      selectedAgentId: null,
      expandedPaths: ["docs/guides"],
    });

    expect(expandedView.entries.map((entry) => entry.path)).toEqual([
      "README.md",
      "docs",
      "docs/guides",
      "docs/guides/deep",
      "docs/guides/setup.md",
    ]);
    expect(
      expandedView.entries.find((entry) => entry.path === "docs/guides/deep")
        ?.expanded,
    ).toBe(false);
  });

  it("narrows the tree to the selected agent's touched files and ancestors", () => {
    const view = buildRepositoryTreeView({
      paths: [
        "cmd",
        "cmd/relay",
        "cmd/relay/main.go",
        "docs",
        "docs/review.md",
        "README.md",
      ],
      touchedFiles: [
        {
          run_id: "run_1",
          agent_id: "agent_coder_1",
          file_path: "cmd/relay/main.go",
          touch_type: "read",
        },
        {
          run_id: "run_1",
          agent_id: "agent_tester_1",
          file_path: "missing/spec.md",
          touch_type: "proposed",
        },
      ],
      selectedAgentId: "agent_coder_1",
      expandedPaths: [],
    });

    expect(view.entries.map((entry) => entry.path)).toEqual([
      "cmd",
      "cmd/relay",
      "cmd/relay/main.go",
    ]);
    expect(view.entries.find((entry) => entry.path === "cmd/relay/main.go")?.touchKinds).toEqual([
      "read",
    ]);
    expect(view.missingTouchedPaths).toEqual([]);
  });

  it("returns an empty filtered tree when the selected agent has no touched files", () => {
    const view = buildRepositoryTreeView({
      paths: ["src", "src/app.ts", "README.md"],
      touchedFiles: [
        {
          run_id: "run_2",
          agent_id: "agent_coder_1",
          file_path: "src/app.ts",
          touch_type: "proposed",
        },
      ],
      selectedAgentId: "agent_reviewer_1",
      expandedPaths: [],
    });

    expect(view.entries).toEqual([]);
    expect(view.missingTouchedPaths).toEqual([]);
  });

  it("shows missing historical touched paths when the repository tree has drifted", () => {
    const view = buildRepositoryTreeView({
      paths: ["src", "src/app.ts"],
      touchedFiles: [
        {
          run_id: "run_2",
          agent_id: "agent_coder_1",
          file_path: "src/app.ts",
          touch_type: "proposed",
        },
        {
          run_id: "run_2",
          agent_id: "agent_coder_1",
          file_path: "src/removed.ts",
          touch_type: "read",
        },
      ],
      selectedAgentId: null,
      expandedPaths: [],
    });

    expect(view.missingTouchedPaths).toEqual(["src/removed.ts"]);
  });

  it("deduplicates touch kinds for the same workspace-wide file path", () => {
    const view = buildRepositoryTreeView({
      paths: ["README.md"],
      touchedFiles: [
        {
          run_id: "run_3",
          agent_id: "agent_coder_1",
          file_path: "README.md",
          touch_type: "read",
        },
        {
          run_id: "run_3",
          agent_id: "agent_coder_1",
          file_path: "README.md",
          touch_type: "read",
        },
        {
          run_id: "run_3",
          agent_id: "agent_reviewer_1",
          file_path: "README.md",
          touch_type: "proposed",
        },
      ],
      selectedAgentId: null,
      expandedPaths: [],
    });

    expect(view.entries).toHaveLength(1);
    expect(view.entries[0]?.touchKinds).toEqual(["proposed", "read"]);
  });
});
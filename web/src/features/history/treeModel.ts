import type { TouchedFilePayload } from "@/shared/lib/workspace-protocol";

export interface RepositoryTreeViewEntry {
  path: string;
  name: string;
  depth: number;
  kind: "directory" | "file";
  expanded: boolean;
  touchKinds: Array<"read" | "proposed">;
}

export interface RepositoryTreeView {
  entries: RepositoryTreeViewEntry[];
  missingTouchedPaths: string[];
}

export function buildRepositoryTreeView(options: {
  paths: string[];
  touchedFiles: TouchedFilePayload[];
  selectedAgentId: string | null;
  expandedPaths: string[];
}): RepositoryTreeView {
  const normalizedPaths = Array.from(
    new Set(options.paths.map((path) => path.trim()).filter(Boolean)),
  ).sort();
  const directoryPaths = new Set<string>();
  for (const path of normalizedPaths) {
    const segments = path.split("/");
    for (let index = 1; index < segments.length; index += 1) {
      directoryPaths.add(segments.slice(0, index).join("/"));
    }
  }
  for (const path of normalizedPaths) {
    if (normalizedPaths.some((candidate) => candidate.startsWith(`${path}/`))) {
      directoryPaths.add(path);
    }
  }

  const filteredTouchedFiles = options.selectedAgentId
    ? options.touchedFiles.filter(
        (item) => item.agent_id === options.selectedAgentId,
      )
    : options.touchedFiles;
  const filteredVisiblePaths = options.selectedAgentId
    ? buildFilteredVisiblePaths(normalizedPaths, filteredTouchedFiles)
    : null;
  const touchMap = new Map<string, Set<"read" | "proposed">>();
  for (const item of filteredTouchedFiles) {
    const nextKinds = touchMap.get(item.file_path) ?? new Set();
    nextKinds.add(item.touch_type);
    touchMap.set(item.file_path, nextKinds);
  }

  const expandedPaths = new Set(options.expandedPaths);
  const entries: RepositoryTreeViewEntry[] = [];
  for (const path of normalizedPaths) {
    if (filteredVisiblePaths && !filteredVisiblePaths.has(path)) {
      continue;
    }

    const depth = path.split("/").length - 1;
    const ancestors = parentPaths(path);
    const visible = options.selectedAgentId
      ? true
      : depth < 2 ||
        ancestors.every(
          (ancestor) =>
            ancestor.split("/").length - 1 === 0 || expandedPaths.has(ancestor),
        );
    if (!visible) {
      continue;
    }

    const kind = directoryPaths.has(path) ? "directory" : "file";
    const touchKinds = Array.from(touchMap.get(path) ?? []).sort((left, right) =>
      left.localeCompare(right),
    );
    entries.push({
      path,
      name: path.split("/").at(-1) ?? path,
      depth,
      kind,
      expanded:
        kind === "directory" &&
        (options.selectedAgentId ? true : depth < 1 || expandedPaths.has(path)),
      touchKinds,
    });
  }

  const missingTouchedPaths = Array.from(touchMap.keys())
    .filter((path) => !normalizedPaths.includes(path))
    .sort((left, right) => left.localeCompare(right));

  return { entries, missingTouchedPaths };
}

function parentPaths(path: string) {
  const segments = path.split("/");
  const parents: string[] = [];
  for (let index = 1; index < segments.length; index += 1) {
    parents.push(segments.slice(0, index).join("/"));
  }
  return parents;
}

function buildFilteredVisiblePaths(
  normalizedPaths: string[],
  touchedFiles: TouchedFilePayload[],
) {
  const visiblePaths = new Set<string>();
  const availablePaths = new Set(normalizedPaths);

  for (const touchedFile of touchedFiles) {
    const filePath = touchedFile.file_path.trim();
    if (!availablePaths.has(filePath)) {
      continue;
    }

    visiblePaths.add(filePath);
    for (const parentPath of parentPaths(filePath)) {
      visiblePaths.add(parentPath);
    }
  }

  return visiblePaths;
}
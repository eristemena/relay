export type RepositoryGraphStatus = "idle" | "loading" | "ready" | "error";

export interface RepositoryGraphNode {
	id: string;
	label: string;
	kind: "directory" | "file";
}

export interface RepositoryGraphEdge {
	id: string;
	source: string;
	target: string;
	strength?: number;
}

export interface RepositoryGraphSnapshot {
	status: RepositoryGraphStatus;
	nodes: RepositoryGraphNode[];
	edges: RepositoryGraphEdge[];
	errorMessage?: string;
}

export const emptyRepositoryGraph: RepositoryGraphSnapshot = {
	status: "idle",
	nodes: [],
	edges: [],
};
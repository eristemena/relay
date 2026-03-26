"use client";

import { useEffect, useId, useRef, useState } from "react";

interface MonacoDiffViewerProps {
	originalContent: string;
	proposedContent: string;
	targetPath: string;
}

type ViewerStatus = "loading" | "ready" | "error";

type MonacoEditorModule = typeof import("monaco-editor/esm/vs/editor/editor.api");
type MonacoDiffEditor = import("monaco-editor/esm/vs/editor/editor.api").editor.IStandaloneDiffEditor;
type MonacoTextModel = import("monaco-editor/esm/vs/editor/editor.api").editor.ITextModel;

type MonacoEnvironmentWindow = Window & {
	MonacoEnvironment?: {
		getWorker?: (workerId: string, label: string) => Worker;
	};
};

function ensureMonacoEnvironment() {
	if (typeof window === "undefined") {
		return;
	}

	const monacoWindow = window as MonacoEnvironmentWindow;

	if (monacoWindow.MonacoEnvironment?.getWorker) {
		return;
	}

	monacoWindow.MonacoEnvironment = {
		getWorker: (_workerId: string, _label: string) =>
			new Worker(
				new URL("monaco-editor/esm/vs/editor/editor.worker.js", import.meta.url),
				{ type: "module" },
			),
	};
}

function languageFromPath(targetPath: string) {
	const normalizedPath = targetPath.toLowerCase();
	if (normalizedPath.endsWith(".ts") || normalizedPath.endsWith(".tsx")) {
		return "typescript";
	}
	if (normalizedPath.endsWith(".js") || normalizedPath.endsWith(".jsx") || normalizedPath.endsWith(".mjs")) {
		return "javascript";
	}
	if (normalizedPath.endsWith(".json")) {
		return "json";
	}
	if (normalizedPath.endsWith(".md")) {
		return "markdown";
	}
	if (normalizedPath.endsWith(".go")) {
		return "go";
	}
	if (normalizedPath.endsWith(".css")) {
		return "css";
	}
	if (normalizedPath.endsWith(".html")) {
		return "html";
	}
	if (normalizedPath.endsWith(".yaml") || normalizedPath.endsWith(".yml")) {
		return "yaml";
	}
	return "plaintext";
}

export function MonacoDiffViewer({
	originalContent,
	proposedContent,
	targetPath,
}: MonacoDiffViewerProps) {
	const containerRef = useRef<HTMLDivElement | null>(null);
	const [status, setStatus] = useState<ViewerStatus>("loading");
	const statusId = useId();

	useEffect(() => {
		let cancelled = false;
		let resizeObserver: ResizeObserver | null = null;
		let diffEditor: MonacoDiffEditor | null = null;
		let originalModel: MonacoTextModel | null = null;
		let modifiedModel: MonacoTextModel | null = null;

		async function loadEditor() {
			if (!containerRef.current) {
				return;
			}

			setStatus("loading");

			try {
				ensureMonacoEnvironment();
				const moduleSpecifier = "monaco-editor/esm/vs/editor/editor.api";
				const monaco = (await import(
					/* @vite-ignore */ moduleSpecifier
				)) as MonacoEditorModule;

				if (cancelled || !containerRef.current) {
					return;
				}

				const language = languageFromPath(targetPath);
				originalModel = monaco.editor.createModel(originalContent, language);
				modifiedModel = monaco.editor.createModel(proposedContent, language);
				diffEditor = monaco.editor.createDiffEditor(containerRef.current, {
					automaticLayout: true,
					glyphMargin: false,
					minimap: { enabled: false },
					readOnly: true,
					renderOverviewRuler: false,
					renderSideBySide: containerRef.current.clientWidth >= 960,
					scrollBeyondLastLine: false,
					theme: "vs-dark",
				});
				if (!diffEditor) {
					return;
				}
				diffEditor.setModel({
					modified: modifiedModel,
					original: originalModel,
				});

				resizeObserver = new ResizeObserver((entries) => {
					const width = entries[0]?.contentRect.width ?? 0;
					diffEditor?.updateOptions({ renderSideBySide: width >= 960 });
					diffEditor?.layout();
				});
				resizeObserver.observe(containerRef.current);

				setStatus("ready");
			} catch {
				if (!cancelled) {
					setStatus("error");
				}
			}
		}

		void loadEditor();

		return () => {
			cancelled = true;
			resizeObserver?.disconnect();
			diffEditor?.dispose();
			originalModel?.dispose();
			modifiedModel?.dispose();
		};
	}, [originalContent, proposedContent, targetPath]);

	return (
		<div className="space-y-3">
			<p className="text-xs text-text-muted" id={statusId}>
				{status === "loading"
					? "Loading Monaco diff review."
					: status === "ready"
						? "Monaco diff review is ready."
						: "Monaco diff review is unavailable, so Relay is showing a plain-text fallback."}
			</p>
			<div className="approval-diff-shell">
				<div
					aria-describedby={statusId}
					aria-label={`Diff review for ${targetPath}`}
					className="approval-monaco-diff"
					ref={containerRef}
					role="img"
				/>
				{status !== "ready" ? (
					<div className="approval-diff-fallback" data-status={status}>
						<div className="grid gap-3 md:grid-cols-2">
							<section
								aria-labelledby={`${statusId}-original`}
								className="rounded-[1.25rem] border border-border bg-base p-3"
							>
								<h5 id={`${statusId}-original`} className="text-sm font-medium text-text">
									Current content
								</h5>
								<pre className="mt-3 overflow-x-auto whitespace-pre-wrap break-words rounded-xl border border-border bg-surface p-3 font-mono text-xs text-text">
									{originalContent || "(new file)"}
								</pre>
							</section>
							<section
								aria-labelledby={`${statusId}-proposed`}
								className="rounded-[1.25rem] border border-brand-mid bg-base p-3"
							>
								<h5 id={`${statusId}-proposed`} className="text-sm font-medium text-text">
									Proposed content
								</h5>
								<pre className="mt-3 overflow-x-auto whitespace-pre-wrap break-words rounded-xl border border-border bg-surface p-3 font-mono text-xs text-text">
									{proposedContent}
								</pre>
							</section>
						</div>
					</div>
				) : null}
			</div>
		</div>
	);
}
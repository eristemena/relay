import { WorkspaceShell } from "@/features/workspace-shell/WorkspaceShell";

export default function HomePage() {
  return (
    <main id="maincontent" tabIndex={-1} className="min-h-screen">
      <WorkspaceShell />
    </main>
  );
}

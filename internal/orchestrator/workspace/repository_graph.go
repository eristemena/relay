package workspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/erisristemena/relay/internal/config"
	"github.com/erisristemena/relay/internal/repository"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

const (
	repositoryGraphStatusIdle    = "idle"
	repositoryGraphStatusLoading = "loading"
	repositoryGraphStatusReady   = "ready"
	repositoryGraphStatusError   = "error"
)

// RepositoryGraphNode is a graph node derived from a repository file relationship scan.
type RepositoryGraphNode = repositoryGraphNode

// RepositoryGraphEdge is a graph edge derived from a repository file relationship scan.
type RepositoryGraphEdge = repositoryGraphEdge

// RepositoryGraphState is the current repository graph status exposed to workspace consumers.
type RepositoryGraphState struct {
	RepositoryRoot string                `json:"repository_root,omitempty"`
	Status         string                `json:"status"`
	Message        string                `json:"message,omitempty"`
	Nodes          []RepositoryGraphNode `json:"nodes,omitempty"`
	Edges          []RepositoryGraphEdge `json:"edges,omitempty"`
}

type repositoryGraphBuilder func(ctx context.Context, repositoryRoot string) (repositoryGraphSnapshot, error)

type repositoryGraphSignatureFunc func(repositoryRoot string) (string, error)

type repositoryGraphCacheEntry struct {
	RepositoryRoot      string
	RepositorySignature string
	Status              string
	Snapshot            repositoryGraphSnapshot
	GeneratedAt         *time.Time
	ErrorMessage        string
}

type repositoryGraphNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
}

type repositoryGraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

type repositoryGraphSnapshot struct {
	RepositoryRoot string                `json:"repository_root"`
	Signature      string                `json:"signature"`
	Nodes          []repositoryGraphNode `json:"nodes"`
	Edges          []repositoryGraphEdge `json:"edges"`
}

func defaultRepositoryGraphSignature(repositoryRoot string) (string, error) {
	repo, err := git.PlainOpen(repositoryRoot)
	if err != nil {
		return "", fmt.Errorf("open repository: %w", err)
	}

	headValue := "unborn"
	head, err := repo.Head()
	if err == nil {
		headValue = head.Hash().String()
	} else if !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return "", fmt.Errorf("read repository head: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("open repository worktree: %w", err)
	}
	status, err := worktree.Status()
	if err != nil {
		return "", fmt.Errorf("read repository worktree status: %w", err)
	}

	paths := make([]string, 0, len(status))
	for path := range status {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	builder := strings.Builder{}
	builder.WriteString(headValue)
	for _, path := range paths {
		fileStatus := status[path]
		builder.WriteString("\n")
		builder.WriteString(path)
		builder.WriteString(":")
		builder.WriteByte(byte(fileStatus.Staging))
		builder.WriteByte(byte(fileStatus.Worktree))
	}

	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:]), nil
}

func defaultRepositoryGraphBuilder(ctx context.Context, repositoryRoot string) (repositoryGraphSnapshot, error) {
	nodes := make(map[string]repositoryGraphNode)
	edges := make(map[string]repositoryGraphEdge)

	err := filepath.WalkDir(repositoryRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		name := entry.Name()
		if entry.IsDir() {
			if name == ".git" || name == "node_modules" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if !isRepositoryGraphFile(path) {
			return nil
		}

		relPath, err := filepath.Rel(repositoryRoot, path)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)
		nodes[relPath] = repositoryGraphNode{ID: relPath, Label: relPath, Kind: "file"}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		for _, importPath := range extractRepositoryGraphImports(relPath, string(content)) {
			resolved, ok := resolveRepositoryGraphImport(repositoryRoot, relPath, importPath)
			if !ok {
				continue
			}
			nodes[resolved] = repositoryGraphNode{ID: resolved, Label: resolved, Kind: "file"}
			edgeID := relPath + "->" + resolved
			edges[edgeID] = repositoryGraphEdge{ID: edgeID, Source: relPath, Target: resolved, Kind: "import"}
		}

		return nil
	})
	if err != nil {
		return repositoryGraphSnapshot{}, err
	}

	nodeList := make([]repositoryGraphNode, 0, len(nodes))
	for _, node := range nodes {
		nodeList = append(nodeList, node)
	}
	sort.Slice(nodeList, func(i, j int) bool {
		return nodeList[i].ID < nodeList[j].ID
	})

	edgeList := make([]repositoryGraphEdge, 0, len(edges))
	for _, edge := range edges {
		edgeList = append(edgeList, edge)
	}
	sort.Slice(edgeList, func(i, j int) bool {
		return edgeList[i].ID < edgeList[j].ID
	})

	return repositoryGraphSnapshot{
		RepositoryRoot: repositoryRoot,
		Nodes:          nodeList,
		Edges:          edgeList,
	}, nil
}

func isRepositoryGraphFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
		return true
	default:
		return false
	}
}

func extractRepositoryGraphImports(relativePath string, content string) []string {
	imports := make([]string, 0)
	seen := make(map[string]struct{})
	add := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		seen[trimmed] = struct{}{}
		imports = append(imports, trimmed)
	}

	if strings.EqualFold(filepath.Ext(relativePath), ".go") {
		inBlock := false
		for _, line := range strings.Split(content, "\n") {
			trimmed := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(trimmed, "import ("):
				inBlock = true
			case inBlock && trimmed == ")":
				inBlock = false
			case strings.HasPrefix(trimmed, "import "):
				if quoted := firstDoubleQuotedValue(trimmed); quoted != "" {
					add(quoted)
				}
			case inBlock:
				if quoted := firstDoubleQuotedValue(trimmed); quoted != "" {
					add(quoted)
				}
			}
		}
		return imports
	}

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if value := quotedPathAfterToken(trimmed, "from"); value != "" {
			add(value)
		}
		if value := quotedPathAfterPrefix(trimmed, "import("); value != "" {
			add(value)
		}
		if value := quotedPathAfterPrefix(trimmed, "import "); value != "" {
			add(value)
		}
		if value := quotedPathAfterToken(trimmed, "require("); value != "" {
			add(value)
		}
	}

	return imports
}

func firstDoubleQuotedValue(line string) string {
	start := strings.IndexByte(line, '"')
	if start < 0 {
		return ""
	}
	end := strings.IndexByte(line[start+1:], '"')
	if end < 0 {
		return ""
	}
	return line[start+1 : start+1+end]
}

func quotedPathAfterToken(line string, token string) string {
	index := strings.Index(line, token)
	if index < 0 {
		return ""
	}
	return quotedPathAfterPrefix(line[index+len(token):], "")
}

func quotedPathAfterPrefix(line string, prefix string) string {
	trimmed := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if trimmed == "" {
		return ""
	}
	quote := trimmed[0]
	if quote != '\'' && quote != '"' {
		return ""
	}
	end := strings.IndexByte(trimmed[1:], quote)
	if end < 0 {
		return ""
	}
	return trimmed[1 : 1+end]
}

func resolveRepositoryGraphImport(repositoryRoot string, importerPath string, importPath string) (string, bool) {
	if !strings.HasPrefix(importPath, ".") {
		return "", false
	}

	baseDir := filepath.Dir(filepath.Join(repositoryRoot, filepath.FromSlash(importerPath)))
	resolvedBase := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(importPath)))
	candidates := []string{resolvedBase}
	for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".go"} {
		candidates = append(candidates, resolvedBase+ext)
		candidates = append(candidates, filepath.Join(resolvedBase, "index"+ext))
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		relPath, err := filepath.Rel(repositoryRoot, candidate)
		if err != nil {
			return "", false
		}
		return filepath.ToSlash(relPath), true
	}

	return "", false
}

func (s *Service) syncRepositoryGraph(ctx context.Context, cfg config.Config) {
	_ = ctx
	rootStatus := repository.ValidateRoot(cfg.ProjectRoot)
	keepRoot := ""
	if rootStatus.Valid {
		keepRoot = rootStatus.Root
	}
	s.pruneRepositoryGraphs(keepRoot)
	if keepRoot == "" {
		return
	}

	signature, err := s.repositoryGraphSignature(keepRoot)
	if err != nil {
		s.storeRepositoryGraphError(keepRoot, "", repositoryGraphBuildErrorMessage())
		return
	}

	if s.hasFreshRepositoryGraph(keepRoot, signature) {
		return
	}

	s.startRepositoryGraphBuild(keepRoot, signature)
}

func (s *Service) pruneRepositoryGraphs(keepRoot string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for root, cancel := range s.repositoryGraphBuilds {
		if root == keepRoot {
			continue
		}
		cancel()
		delete(s.repositoryGraphBuilds, root)
	}
	for root := range s.repositoryGraphs {
		if root == keepRoot {
			continue
		}
		delete(s.repositoryGraphs, root)
	}
}

func (s *Service) hasFreshRepositoryGraph(repositoryRoot string, signature string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.repositoryGraphs[repositoryRoot]
	if !ok || entry.RepositorySignature != signature {
		return false
	}
	return entry.Status == repositoryGraphStatusLoading || entry.Status == repositoryGraphStatusReady || entry.Status == repositoryGraphStatusError
}

func (s *Service) startRepositoryGraphBuild(repositoryRoot string, signature string) {
	buildCtx, cancel := context.WithCancel(context.Background())

	s.mu.Lock()
	if existingCancel, ok := s.repositoryGraphBuilds[repositoryRoot]; ok {
		existingCancel()
	}
	s.repositoryGraphBuilds[repositoryRoot] = cancel
	s.repositoryGraphs[repositoryRoot] = repositoryGraphCacheEntry{
		RepositoryRoot:      repositoryRoot,
		RepositorySignature: signature,
		Status:              repositoryGraphStatusLoading,
		Snapshot: repositoryGraphSnapshot{
			RepositoryRoot: repositoryRoot,
			Signature:      signature,
		},
	}
	builder := s.repositoryGraphBuilder
	s.mu.Unlock()

	go func() {
		snapshot, err := builder(buildCtx, repositoryRoot)
		if err != nil {
			if buildCtx.Err() != nil {
				return
			}
			s.storeRepositoryGraphError(repositoryRoot, signature, repositoryGraphBuildErrorMessage())
			return
		}

		now := time.Now().UTC()
		snapshot.RepositoryRoot = repositoryRoot
		snapshot.Signature = signature

		s.mu.Lock()
		entry, ok := s.repositoryGraphs[repositoryRoot]
		if !ok || entry.RepositorySignature != signature {
			s.mu.Unlock()
			return
		}
		delete(s.repositoryGraphBuilds, repositoryRoot)
		entry = repositoryGraphCacheEntry{
			RepositoryRoot:      repositoryRoot,
			RepositorySignature: signature,
			Status:              repositoryGraphStatusReady,
			Snapshot:            snapshot,
			GeneratedAt:         &now,
		}
		s.repositoryGraphs[repositoryRoot] = entry
		readyState := repositoryGraphStateFromConnection(ConnectedRepositorySummary{Path: repositoryRoot, Status: "connected"}, entry, true)
		s.mu.Unlock()
		s.dispatchWorkspaceEnvelope(StreamEnvelope{Type: "repository_graph_status", Payload: readyState})
	}()
}

func (s *Service) storeRepositoryGraphError(repositoryRoot string, signature string, message string) {
	s.mu.Lock()
	delete(s.repositoryGraphBuilds, repositoryRoot)
	entry := s.repositoryGraphs[repositoryRoot]
	entry.RepositoryRoot = repositoryRoot
	if signature != "" {
		entry.RepositorySignature = signature
	}
	entry.Status = repositoryGraphStatusError
	entry.ErrorMessage = strings.TrimSpace(message)
	entry.GeneratedAt = nil
	entry.Snapshot = repositoryGraphSnapshot{RepositoryRoot: repositoryRoot, Signature: entry.RepositorySignature}
	s.repositoryGraphs[repositoryRoot] = entry
	errorState := repositoryGraphStateFromConnection(ConnectedRepositorySummary{Path: repositoryRoot, Status: "connected"}, entry, true)
	s.mu.Unlock()
	s.dispatchWorkspaceEnvelope(StreamEnvelope{Type: "repository_graph_status", Payload: errorState})
}

func repositoryGraphBuildErrorMessage() string {
	return "Relay could not build the repository graph yet. The rest of the workspace remains available."
}

func repositoryGraphStateFromConnection(connected ConnectedRepositorySummary, entry repositoryGraphCacheEntry, ok bool) RepositoryGraphState {
	if ok {
		message := strings.TrimSpace(entry.ErrorMessage)
		switch entry.Status {
		case repositoryGraphStatusReady:
			return RepositoryGraphState{
				RepositoryRoot: entry.RepositoryRoot,
				Status:         repositoryGraphStatusReady,
				Message:        "Repository graph ready.",
				Nodes:          entry.Snapshot.Nodes,
				Edges:          entry.Snapshot.Edges,
			}
		case repositoryGraphStatusError:
			if message == "" {
				message = repositoryGraphBuildErrorMessage()
			}
			return RepositoryGraphState{
				RepositoryRoot: entry.RepositoryRoot,
				Status:         repositoryGraphStatusError,
				Message:        message,
			}
		case repositoryGraphStatusLoading:
			return RepositoryGraphState{
				RepositoryRoot: entry.RepositoryRoot,
				Status:         repositoryGraphStatusLoading,
				Message:        "Building repository graph in the background.",
			}
		}
	}

	switch connected.Status {
	case "connected":
		return RepositoryGraphState{
			RepositoryRoot: connected.Path,
			Status:         repositoryGraphStatusLoading,
			Message:        "Building repository graph in the background.",
		}
	case "invalid":
		message := strings.TrimSpace(connected.Message)
		if message == "" {
			message = repositoryGraphBuildErrorMessage()
		}
		return RepositoryGraphState{
			RepositoryRoot: connected.Path,
			Status:         repositoryGraphStatusError,
			Message:        message,
		}
	default:
		return RepositoryGraphState{
			Status:  repositoryGraphStatusIdle,
			Message: "Connect a repository to load the background-built codebase graph.",
		}
	}
}

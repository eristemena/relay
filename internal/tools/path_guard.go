package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

func resolveWithinRoot(projectRoot string, requestedPath string) (string, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return "", fmt.Errorf("Repository-reading tools are unavailable until you set a valid project_root in Relay's local configuration.")
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve project_root: %w", err)
	}
	target := requestedPath
	if strings.TrimSpace(target) == "" || target == "." {
		return absoluteRoot, nil
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(absoluteRoot, target)
	}
	absoluteTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	if absoluteTarget != absoluteRoot && !strings.HasPrefix(absoluteTarget, absoluteRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("Relay blocked access outside the configured project_root")
	}
	return absoluteTarget, nil
}
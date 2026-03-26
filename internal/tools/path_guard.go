package tools

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/erisristemena/relay/internal/repository"
)

func resolveWithinRoot(projectRoot string, requestedPath string) (string, error) {
	rootStatus := repository.ValidateRoot(projectRoot)
	if !rootStatus.Valid {
		return "", errors.New(strings.TrimSpace(rootStatus.Message))
	}
	absoluteRoot := rootStatus.Root
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

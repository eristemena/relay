package browser

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

type Launcher interface {
	Open(ctx context.Context, targetURL string) error
}

type ExecLauncher struct{}

func (ExecLauncher) Open(ctx context.Context, targetURL string) error {
	command, args, err := commandForOS(targetURL)
	if err != nil {
		return err
	}

	if err := exec.CommandContext(ctx, command, args...).Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}

	return nil
}

func commandForOS(targetURL string) (string, []string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{targetURL}, nil
	case "linux":
		return "xdg-open", []string{targetURL}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", targetURL}, nil
	default:
		return "", nil, fmt.Errorf("unsupported platform for automatic browser launch: %s", runtime.GOOS)
	}
}

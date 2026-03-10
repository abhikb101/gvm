package platform

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser opens a URL in the user's default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("browser open not supported on %s — visit %s manually", runtime.GOOS, url)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("opening browser: %w", err)
	}
	return nil
}

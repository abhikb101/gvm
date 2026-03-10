package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gvm-tools/gvm/internal/profile"
)

// CloneWithProfile clones a repository using the given profile's credentials.
// Returns the path to the cloned directory.
func CloneWithProfile(p *profile.Profile, repoURL string, targetDir string) (string, error) {
	// Determine the clone directory name
	if targetDir == "" {
		targetDir = repoNameFromURL(repoURL)
	}

	args := []string{"clone"}

	// Set up auth-specific clone args
	if isSSHURL(repoURL) && p.HasSSH() && p.SSHKeyPath != "" {
		sshCmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes -o IdentityAgent=none", p.SSHKeyPath)
		args = append(args, "-c", fmt.Sprintf("core.sshCommand=%s", sshCmd))
	}

	args = append(args, repoURL, targetDir)

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// For HTTPS with token, set credential helper via env
	if isHTTPSURL(repoURL) && p.HasHTTP() {
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("GIT_CONFIG_VALUE_0=!gvm _credential-helper %s", p.Name),
			"GIT_CONFIG_KEY_0=credential.https://github.com.helper",
			"GIT_CONFIG_COUNT=1",
		)
	}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("cloning repository: %w", err)
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return targetDir, nil
	}
	return absPath, nil
}

// DetectURLAuthMismatch checks if the URL format matches the profile's auth method.
func DetectURLAuthMismatch(repoURL string, p *profile.Profile) string {
	if isSSHURL(repoURL) && !p.HasSSH() {
		return fmt.Sprintf("URL is SSH but profile '%s' only has HTTP auth. Consider using an HTTPS URL instead.", p.Name)
	}
	if isHTTPSURL(repoURL) && !p.HasHTTP() {
		return fmt.Sprintf("URL is HTTPS but profile '%s' only has SSH auth. Consider using an SSH URL instead.", p.Name)
	}
	return ""
}

func isSSHURL(url string) bool {
	return strings.HasPrefix(url, "git@") || strings.HasPrefix(url, "ssh://")
}

func isHTTPSURL(url string) bool {
	return strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://")
}

func repoNameFromURL(url string) string {
	// Handle git@github.com:user/repo.git
	if idx := strings.LastIndex(url, "/"); idx >= 0 {
		name := url[idx+1:]
		name = strings.TrimSuffix(name, ".git")
		return name
	}
	if idx := strings.LastIndex(url, ":"); idx >= 0 {
		name := url[idx+1:]
		name = strings.TrimSuffix(name, ".git")
		// Handle user/repo format
		if slashIdx := strings.LastIndex(name, "/"); slashIdx >= 0 {
			name = name[slashIdx+1:]
		}
		return name
	}
	return url
}

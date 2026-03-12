package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// VerifySSHConnection tests an SSH connection to GitHub using the given key.
// Returns nil if the connection succeeds and the username matches.
func VerifySSHConnection(keyPath, expectedUsername string) error {
	cmd := exec.Command("ssh",
		"-T",
		"-F", "/dev/null",
		"-i", keyPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"git@github.com",
	)

	// Disable the SSH agent for this command so only the specified key is tried.
	// Without this, agent-loaded keys from other profiles can interfere.
	env := os.Environ()
	filtered := env[:0]
	for _, e := range env {
		if !strings.HasPrefix(e, "SSH_AUTH_SOCK=") {
			filtered = append(filtered, e)
		}
	}
	cmd.Env = append(filtered, "SSH_AUTH_SOCK=")

	// GitHub returns exit code 1 even on successful auth
	output, _ := cmd.CombinedOutput()
	result := string(output)

	if strings.Contains(result, "Hi "+expectedUsername+"!") {
		return nil
	}

	// Check for common SSH errors
	if strings.Contains(result, "Permission denied") {
		return fmt.Errorf("permission denied — the SSH key may not be added to your GitHub account yet")
	}
	if strings.Contains(result, "Connection refused") || strings.Contains(result, "Connection timed out") {
		return fmt.Errorf("cannot connect to GitHub — check your internet connection")
	}

	// Authenticated but as the wrong user
	if strings.Contains(result, "Hi ") && strings.Contains(result, "successfully authenticated") {
		actual := result[strings.Index(result, "Hi ")+3:]
		if idx := strings.Index(actual, "!"); idx > 0 {
			actual = actual[:idx]
		}
		return fmt.Errorf("SSH key is registered to GitHub user '%s', not '%s' — make sure you added the key to the correct GitHub account", actual, expectedUsername)
	}

	return fmt.Errorf("SSH verification failed — expected user '%s', got: %s", expectedUsername, strings.TrimSpace(result))
}

// VerifyGitHubToken validates a GitHub OAuth token and returns the associated username.
func VerifyGitHubToken(token string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gvm-cli")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("connecting to GitHub API (check your internet): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("token expired or revoked")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var user struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("parsing GitHub response: %w", err)
	}

	return user.Login, nil
}

// SSHConnectionStatus checks SSH auth status for a profile.
// Returns "connected", "failed", or an error description.
func SSHConnectionStatus(keyPath, username string) string {
	if err := VerifySSHConnection(keyPath, username); err != nil {
		return "failed"
	}
	return "connected"
}

// TokenStatus checks an OAuth token's validity.
// Returns "connected", "expired", or "not configured".
func TokenStatus(token string) string {
	if token == "" {
		return "not configured"
	}
	_, err := VerifyGitHubToken(token)
	if err != nil {
		if strings.Contains(err.Error(), "expired") || strings.Contains(err.Error(), "revoked") {
			return "expired"
		}
		return "failed"
	}
	return "connected"
}

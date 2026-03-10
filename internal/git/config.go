package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// SetLocalConfig sets a git config value at the repository level.
func SetLocalConfig(key, value string) error {
	return gitConfig("--local", key, value)
}

// SetGlobalConfig sets a git config value at the user level (~/.gitconfig).
func SetGlobalConfig(key, value string) error {
	return gitConfig("--global", key, value)
}

// GetLocalConfig reads a git config value at the repository level.
func GetLocalConfig(key string) (string, error) {
	return gitConfigGet("--local", key)
}

// GetGlobalConfig reads a git config value at the user level.
func GetGlobalConfig(key string) (string, error) {
	return gitConfigGet("--global", key)
}

// UnsetLocalConfig removes a git config value at the repository level.
func UnsetLocalConfig(key string) error {
	cmd := exec.Command("git", "config", "--local", "--unset", key)
	cmd.Run() // best effort
	return nil
}

// ConfigureIdentity sets user.name, user.email, and optionally core.sshCommand.
func ConfigureIdentity(scope string, name, email, sshKeyPath string) error {
	flag := "--" + scope

	if err := gitConfig(flag, "user.name", name); err != nil {
		return fmt.Errorf("setting user.name: %w", err)
	}
	if err := gitConfig(flag, "user.email", email); err != nil {
		return fmt.Errorf("setting user.email: %w", err)
	}

	if sshKeyPath != "" && scope == "local" {
		sshCmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes -o IdentityAgent=none", sshKeyPath)
		if err := gitConfig(flag, "core.sshCommand", sshCmd); err != nil {
			return fmt.Errorf("setting core.sshCommand: %w", err)
		}
	}

	return nil
}

// ConfigureCredentialHelper sets up GVM as the credential helper for GitHub.
func ConfigureCredentialHelper(scope, profileName string) error {
	flag := "--" + scope
	helper := fmt.Sprintf("!gvm _credential-helper %s", profileName)
	return gitConfig(flag, "credential.https://github.com.helper", helper)
}

func gitConfig(flag, key, value string) error {
	cmd := exec.Command("git", "config", flag, key, value)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config %s %s: %s", flag, key, strings.TrimSpace(string(out)))
	}
	return nil
}

func gitConfigGet(flag, key string) (string, error) {
	cmd := exec.Command("git", "config", flag, key)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

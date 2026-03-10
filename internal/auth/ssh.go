package auth

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gvm-tools/gvm/internal/platform"
	"github.com/gvm-tools/gvm/internal/ui"
)

// SSHKeyPath returns the standard GVM SSH key path for a profile.
func SSHKeyPath(profileName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "gvm_"+profileName), nil
}

// GenerateSSHKey creates an Ed25519 SSH key pair for a profile.
// Returns the path to the private key.
func GenerateSSHKey(profileName string) (string, error) {
	keyPath, err := SSHKeyPath(profileName)
	if err != nil {
		return "", err
	}

	// Ensure ~/.ssh exists
	sshDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return "", fmt.Errorf("creating SSH directory: %w", err)
	}

	// Remove existing key files so ssh-keygen doesn't prompt for overwrite
	os.Remove(keyPath)
	os.Remove(keyPath + ".pub")

	cmd := exec.Command("ssh-keygen",
		"-t", "ed25519",
		"-C", fmt.Sprintf("gvm:%s", profileName),
		"-f", keyPath,
		"-N", "",
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("generating SSH key: %s", strings.TrimSpace(string(out)))
	}

	// Ensure correct permissions
	if err := os.Chmod(keyPath, 0600); err != nil {
		return "", fmt.Errorf("setting key permissions: %w", err)
	}
	if err := os.Chmod(keyPath+".pub", 0644); err != nil {
		return "", fmt.Errorf("setting public key permissions: %w", err)
	}

	return keyPath, nil
}

// SSHKeyExists returns true if an SSH key exists for the given profile.
func SSHKeyExists(profileName string) (bool, error) {
	keyPath, err := SSHKeyPath(profileName)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(keyPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// CopyPublicKeyToClipboard reads the public key and copies it to clipboard.
func CopyPublicKeyToClipboard(profileName string) error {
	keyPath, err := SSHKeyPath(profileName)
	if err != nil {
		return err
	}

	pubKey, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return fmt.Errorf("reading public key: %w", err)
	}

	if err := platform.CopyToClipboard(string(pubKey)); err != nil {
		// Fallback: print the key instead
		ui.Warn("Could not copy to clipboard: %v", err)
		fmt.Println()
		fmt.Println(strings.TrimSpace(string(pubKey)))
		fmt.Println()
		fmt.Println("Copy the key above and add it to GitHub.")
		return nil
	}

	ui.Success("Public key copied to clipboard")
	return nil
}

// AddKeyToAgent adds an SSH key to the running ssh-agent.
func AddKeyToAgent(keyPath string) error {
	if err := ensureSSHAgent(); err != nil {
		return err
	}

	cmd := exec.Command("ssh-add", keyPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("adding key to ssh-agent: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// RemoveKeyFromAgent removes an SSH key from the ssh-agent.
func RemoveKeyFromAgent(keyPath string) error {
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return nil
	}
	cmd := exec.Command("ssh-add", "-d", keyPath)
	cmd.Run() // best effort — ignore errors
	return nil
}

// DeleteSSHKey removes both private and public key files.
func DeleteSSHKey(profileName string) error {
	keyPath, err := SSHKeyPath(profileName)
	if err != nil {
		return err
	}

	RemoveKeyFromAgent(keyPath)

	for _, p := range []string{keyPath, keyPath + ".pub"} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("deleting key %s: %w", p, err)
		}
	}
	return nil
}

func ensureSSHAgent() error {
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		return nil
	}

	// On macOS, the system ssh-agent is typically available via launchd.
	// Try to find its socket before asking the user to start one manually.
	if out, err := exec.Command("launchctl", "getenv", "SSH_AUTH_SOCK").Output(); err == nil {
		if sock := strings.TrimSpace(string(out)); sock != "" {
			os.Setenv("SSH_AUTH_SOCK", sock)
			return nil
		}
	}

	return fmt.Errorf("ssh-agent is not running — start it with: eval $(ssh-agent -s)")
}

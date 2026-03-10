package platform

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

const keychainService = "gvm"

// KeychainAvailable returns true if the OS keychain is accessible.
func KeychainAvailable() bool {
	switch runtime.GOOS {
	case "darwin":
		_, err := exec.LookPath("security")
		return err == nil
	case "linux":
		_, err := exec.LookPath("secret-tool")
		return err == nil
	default:
		return false
	}
}

// KeychainStore saves a secret to the OS keychain.
func KeychainStore(profileName, token string) error {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("security", "add-generic-password",
			"-a", profileName,
			"-s", keychainService,
			"-w", token,
			"-U", // update if exists
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("storing in keychain: %s", strings.TrimSpace(string(out)))
		}
		return nil

	case "linux":
		cmd := exec.Command("secret-tool", "store",
			"--label", fmt.Sprintf("GVM token for %s", profileName),
			"service", keychainService,
			"account", profileName,
		)
		cmd.Stdin = strings.NewReader(token)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("storing in secret storage: %s", strings.TrimSpace(string(out)))
		}
		return nil

	default:
		return fmt.Errorf("keychain not supported on %s", runtime.GOOS)
	}
}

// KeychainLoad retrieves a secret from the OS keychain.
func KeychainLoad(profileName string) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("security", "find-generic-password",
			"-a", profileName,
			"-s", keychainService,
			"-w",
		).Output()
		if err != nil {
			return "", fmt.Errorf("reading from keychain: %w", err)
		}
		return strings.TrimSpace(string(out)), nil

	case "linux":
		out, err := exec.Command("secret-tool", "lookup",
			"service", keychainService,
			"account", profileName,
		).Output()
		if err != nil {
			return "", fmt.Errorf("reading from secret storage: %w", err)
		}
		return strings.TrimSpace(string(out)), nil

	default:
		return "", fmt.Errorf("keychain not supported on %s", runtime.GOOS)
	}
}

// KeychainDelete removes a secret from the OS keychain.
func KeychainDelete(profileName string) error {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("security", "delete-generic-password",
			"-a", profileName,
			"-s", keychainService,
		)
		if err := cmd.Run(); err != nil {
			return nil // not found is fine
		}
		return nil

	case "linux":
		cmd := exec.Command("secret-tool", "clear",
			"service", keychainService,
			"account", profileName,
		)
		if err := cmd.Run(); err != nil {
			return nil
		}
		return nil

	default:
		return nil
	}
}

package cmd

import (
	"time"

	"github.com/gvm-tools/gvm/internal/crypto"
	"github.com/gvm-tools/gvm/internal/platform"
	"github.com/gvm-tools/gvm/internal/profile"
)

// timeNow returns the current UTC time. Extracted for testability.
func timeNow() time.Time {
	return time.Now().UTC()
}

// openGitHubSSHSettings opens the GitHub SSH key settings page.
func openGitHubSSHSettings() {
	_ = platform.OpenBrowser("https://github.com/settings/ssh/new")
}

// storeToken encrypts and stores an OAuth token in the profile.
// Prefers OS keychain if available, falls back to file-level encryption.
func storeToken(p *profile.Profile, token string) error {
	if platform.KeychainAvailable() {
		if err := platform.KeychainStore(p.Name, token); err == nil {
			p.GHTokenEncrypted = "keychain"
			return nil
		}
	}

	encrypted, err := crypto.Encrypt(token)
	if err != nil {
		return err
	}
	p.GHTokenEncrypted = encrypted
	return nil
}

// loadToken retrieves a stored OAuth token for a profile.
func loadToken(p *profile.Profile) (string, error) {
	if p.GHTokenEncrypted == "" {
		return "", nil
	}

	if p.GHTokenEncrypted == "keychain" {
		return platform.KeychainLoad(p.Name)
	}

	return crypto.Decrypt(p.GHTokenEncrypted)
}

// deleteToken removes a stored token from both keychain and profile.
func deleteToken(p *profile.Profile) {
	if p.GHTokenEncrypted == "keychain" {
		_ = platform.KeychainDelete(p.Name)
	}
	p.GHTokenEncrypted = ""
}

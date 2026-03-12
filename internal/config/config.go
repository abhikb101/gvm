package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gvm-tools/gvm/internal/fsutil"
)

const (
	configDirName   = ".gvm"
	configFileName  = "config.json"
	activeFileName  = "active"
	profilesDirName = "profiles"
	configVersion   = "1"
)

// DefaultGitHubClientID is the OAuth App client ID embedded at build time.
// Users can override this via GVM_GITHUB_CLIENT_ID env var or config.
const DefaultGitHubClientID = ""

// Config holds global GVM settings.
type Config struct {
	Version        string `json:"version"`
	DefaultAuth    string `json:"default_auth"`
	AutoSwitch     bool   `json:"auto_switch"`
	PromptDisplay  bool   `json:"prompt_display"`
	Shell          string `json:"shell"`
	Editor         string `json:"editor,omitempty"`
	GitHubClientID string `json:"github_client_id,omitempty"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(shell string) Config {
	return Config{
		Version:       configVersion,
		DefaultAuth:   "ssh",
		AutoSwitch:    true,
		PromptDisplay: true,
		Shell:         shell,
	}
}

// Dir returns the GVM config directory path (~/.gvm).
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, configDirName), nil
}

// ProfilesDir returns the path to the profiles directory (~/.gvm/profiles).
func ProfilesDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, profilesDirName), nil
}

// EnsureDirectories creates the GVM directory structure if it doesn't exist.
func EnsureDirectories() error {
	dir, err := Dir()
	if err != nil {
		return err
	}

	dirs := []string{
		dir,
		filepath.Join(dir, profilesDirName),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}
	return nil
}

// Load reads the global config from disk.
func Load() (*Config, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, configFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("GVM not initialized — run 'gvm init' first")
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Save writes the config to disk atomically.
func (c *Config) Save() error {
	dir, err := Dir()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	data = append(data, '\n')

	return fsutil.AtomicWrite(filepath.Join(dir, configFileName), data, 0644)
}

// GetActive reads the currently active profile name.
func GetActive() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(filepath.Join(dir, activeFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("reading active profile: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// SetActive writes the active profile name to disk.
func SetActive(name string) error {
	dir, err := Dir()
	if err != nil {
		return err
	}

	return fsutil.AtomicWrite(filepath.Join(dir, activeFileName), []byte(name+"\n"), 0644)
}

// ClearActive removes the active profile marker.
func ClearActive() error {
	dir, err := Dir()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, activeFileName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clearing active profile: %w", err)
	}
	return nil
}

// Exists returns true if GVM has been initialized.
func Exists() bool {
	dir, err := Dir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, configFileName))
	return err == nil
}

// GetGitHubClientID returns the effective GitHub OAuth client ID,
// checking env var > config > compiled default.
func (c *Config) GetGitHubClientID() string {
	if envID := os.Getenv("GVM_GITHUB_CLIENT_ID"); envID != "" {
		return envID
	}
	if c.GitHubClientID != "" {
		return c.GitHubClientID
	}
	return DefaultGitHubClientID
}

// Set updates a single config key. Returns an error for unknown keys.
func (c *Config) Set(key, value string) error {
	switch key {
	case "default-auth":
		if value != "ssh" && value != "http" && value != "both" {
			return fmt.Errorf("invalid auth method '%s' — must be ssh, http, or both", value)
		}
		c.DefaultAuth = value
	case "auto-switch":
		c.AutoSwitch = value == "true" || value == "1" || value == "yes"
	case "prompt":
		c.PromptDisplay = value == "true" || value == "1" || value == "yes"
	case "editor":
		c.Editor = value
	case "github-client-id":
		c.GitHubClientID = value
	default:
		return fmt.Errorf("unknown config key '%s' — valid keys: default-auth, auto-switch, prompt, editor, github-client-id", key)
	}
	return nil
}

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestHome(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	return tmpDir
}

func TestEnsureDirectories(t *testing.T) {
	home := setupTestHome(t)

	if err := EnsureDirectories(); err != nil {
		t.Fatalf("EnsureDirectories() error = %v", err)
	}

	// Verify directories exist
	for _, dir := range []string{".gvm", ".gvm/profiles"} {
		path := filepath.Join(home, dir)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("directory %s not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}

	// Idempotent
	if err := EnsureDirectories(); err != nil {
		t.Errorf("second EnsureDirectories() error = %v", err)
	}
}

func TestConfigSaveLoad(t *testing.T) {
	setupTestHome(t)

	if err := EnsureDirectories(); err != nil {
		t.Fatalf("EnsureDirectories() error = %v", err)
	}

	cfg := DefaultConfig("zsh")
	cfg.GitHubClientID = "test-client-id"

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Version != configVersion {
		t.Errorf("Version = %q, want %q", loaded.Version, configVersion)
	}
	if loaded.DefaultAuth != "ssh" {
		t.Errorf("DefaultAuth = %q, want %q", loaded.DefaultAuth, "ssh")
	}
	if !loaded.AutoSwitch {
		t.Error("AutoSwitch = false, want true")
	}
	if !loaded.PromptDisplay {
		t.Error("PromptDisplay = false, want true")
	}
	if loaded.Shell != "zsh" {
		t.Errorf("Shell = %q, want %q", loaded.Shell, "zsh")
	}
	if loaded.GitHubClientID != "test-client-id" {
		t.Errorf("GitHubClientID = %q, want %q", loaded.GitHubClientID, "test-client-id")
	}
}

func TestLoadNotInitialized(t *testing.T) {
	setupTestHome(t)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should return error when not initialized")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("Load() error = %q, want error containing 'not initialized'", err.Error())
	}
}

func TestActiveProfile(t *testing.T) {
	setupTestHome(t)

	if err := EnsureDirectories(); err != nil {
		t.Fatalf("EnsureDirectories() error = %v", err)
	}

	// No active profile initially
	active, err := GetActive()
	if err != nil {
		t.Fatalf("GetActive() error = %v", err)
	}
	if active != "" {
		t.Errorf("GetActive() = %q, want empty", active)
	}

	// Set active
	if err := SetActive("personal"); err != nil {
		t.Fatalf("SetActive() error = %v", err)
	}

	active, err = GetActive()
	if err != nil {
		t.Fatalf("GetActive() error = %v", err)
	}
	if active != "personal" {
		t.Errorf("GetActive() = %q, want %q", active, "personal")
	}

	// Switch active
	if err := SetActive("work"); err != nil {
		t.Fatalf("SetActive() error = %v", err)
	}

	active, err = GetActive()
	if err != nil {
		t.Fatalf("GetActive() error = %v", err)
	}
	if active != "work" {
		t.Errorf("GetActive() = %q, want %q", active, "work")
	}

	// Clear active
	if err := ClearActive(); err != nil {
		t.Fatalf("ClearActive() error = %v", err)
	}

	active, err = GetActive()
	if err != nil {
		t.Fatalf("GetActive() error = %v", err)
	}
	if active != "" {
		t.Errorf("GetActive() after clear = %q, want empty", active)
	}
}

func TestConfigSet(t *testing.T) {
	cfg := DefaultConfig("zsh")

	tests := []struct {
		key     string
		value   string
		wantErr bool
	}{
		{"default-auth", "ssh", false},
		{"default-auth", "http", false},
		{"default-auth", "both", false},
		{"default-auth", "ftp", true},
		{"auto-switch", "true", false},
		{"auto-switch", "false", false},
		{"prompt", "true", false},
		{"prompt", "false", false},
		{"editor", "vim", false},
		{"github-client-id", "Iv1.abc123", false},
		{"unknown-key", "value", true},
	}

	for _, tt := range tests {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			err := cfg.Set(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Set(%q, %q) error = %v, wantErr %v", tt.key, tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestGetGitHubClientID(t *testing.T) {
	cfg := DefaultConfig("zsh")

	// No config, no env — returns compiled default
	id := cfg.GetGitHubClientID()
	if id != DefaultGitHubClientID {
		t.Errorf("GetGitHubClientID() = %q, want %q", id, DefaultGitHubClientID)
	}

	// Config value set
	cfg.GitHubClientID = "config-id"
	id = cfg.GetGitHubClientID()
	if id != "config-id" {
		t.Errorf("GetGitHubClientID() = %q, want %q", id, "config-id")
	}

	// Env var overrides config
	t.Setenv("GVM_GITHUB_CLIENT_ID", "env-id")
	id = cfg.GetGitHubClientID()
	if id != "env-id" {
		t.Errorf("GetGitHubClientID() = %q, want %q", id, "env-id")
	}
}

func TestExists(t *testing.T) {
	setupTestHome(t)

	if Exists() {
		t.Error("Exists() = true before init")
	}

	EnsureDirectories()
	cfg := DefaultConfig("zsh")
	cfg.Save()

	if !Exists() {
		t.Error("Exists() = false after init")
	}
}

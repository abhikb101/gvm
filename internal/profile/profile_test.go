package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gvm-tools/gvm/internal/config"
)

func setupTestDir(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	if err := config.EnsureDirectories(); err != nil {
		t.Fatalf("setting up test dir: %v", err)
	}
}

func TestProfileValidation(t *testing.T) {
	tests := []struct {
		name    string
		profile Profile
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid profile",
			profile: Profile{Name: "personal", GitName: "John", GitEmail: "john@example.com", AuthMethod: AuthSSH},
			wantErr: false,
		},
		{
			name:    "valid profile with hyphens",
			profile: Profile{Name: "my-work-1", GitName: "John", GitEmail: "john@work.com", AuthMethod: AuthHTTP},
			wantErr: false,
		},
		{
			name:    "empty name",
			profile: Profile{Name: "", GitName: "John", GitEmail: "john@example.com", AuthMethod: AuthSSH},
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "invalid name characters",
			profile: Profile{Name: "my profile!", GitName: "John", GitEmail: "john@example.com", AuthMethod: AuthSSH},
			wantErr: true,
			errMsg:  "invalid",
		},
		{
			name:    "name with uppercase",
			profile: Profile{Name: "MyProfile", GitName: "John", GitEmail: "john@example.com", AuthMethod: AuthSSH},
			wantErr: true,
			errMsg:  "invalid",
		},
		{
			name:    "name starting with hyphen",
			profile: Profile{Name: "-personal", GitName: "John", GitEmail: "john@example.com", AuthMethod: AuthSSH},
			wantErr: true,
			errMsg:  "invalid",
		},
		{
			name:    "name too long",
			profile: Profile{Name: strings.Repeat("a", 33), GitName: "John", GitEmail: "john@example.com", AuthMethod: AuthSSH},
			wantErr: true,
			errMsg:  "too long",
		},
		{
			name:    "name exactly max length",
			profile: Profile{Name: strings.Repeat("a", 32), GitName: "John", GitEmail: "john@example.com", AuthMethod: AuthSSH},
			wantErr: false,
		},
		{
			name:    "empty git name",
			profile: Profile{Name: "test", GitName: "", GitEmail: "john@example.com", AuthMethod: AuthSSH},
			wantErr: true,
			errMsg:  "name cannot be empty",
		},
		{
			name:    "whitespace-only git name",
			profile: Profile{Name: "test", GitName: "   ", GitEmail: "john@example.com", AuthMethod: AuthSSH},
			wantErr: true,
			errMsg:  "name cannot be empty",
		},
		{
			name:    "invalid email - no at sign",
			profile: Profile{Name: "test", GitName: "John", GitEmail: "not-an-email", AuthMethod: AuthSSH},
			wantErr: true,
			errMsg:  "invalid email",
		},
		{
			name:    "invalid email - no domain",
			profile: Profile{Name: "test", GitName: "John", GitEmail: "john@", AuthMethod: AuthSSH},
			wantErr: true,
			errMsg:  "invalid email",
		},
		{
			name:    "invalid auth method",
			profile: Profile{Name: "test", GitName: "John", GitEmail: "john@example.com", AuthMethod: "ftp"},
			wantErr: true,
			errMsg:  "invalid auth method",
		},
		{
			name:    "auth method both",
			profile: Profile{Name: "test", GitName: "John", GitEmail: "john@example.com", AuthMethod: AuthBoth},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestProfileSaveLoad(t *testing.T) {
	setupTestDir(t)

	p := &Profile{
		Name:           "test-profile",
		GitName:        "Test User",
		GitEmail:       "test@example.com",
		GitHubUsername: "testuser",
		AuthMethod:     AuthSSH,
		SSHKeyPath:     "~/.ssh/gvm_test",
		CreatedAt:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		LastUsed:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Save
	if err := p.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	dir, _ := config.ProfilesDir()
	path := filepath.Join(dir, "test-profile.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("profile file not created: %v", err)
	}

	// Verify file permissions
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected permissions 0600, got %o", info.Mode().Perm())
	}

	// Load
	loaded, err := Load("test-profile")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Name != p.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, p.Name)
	}
	if loaded.GitEmail != p.GitEmail {
		t.Errorf("GitEmail = %q, want %q", loaded.GitEmail, p.GitEmail)
	}
	if loaded.GitHubUsername != p.GitHubUsername {
		t.Errorf("GitHubUsername = %q, want %q", loaded.GitHubUsername, p.GitHubUsername)
	}
}

func TestProfileExists(t *testing.T) {
	setupTestDir(t)

	// Should not exist initially
	exists, err := Exists("nonexistent")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Exists() = true for nonexistent profile")
	}

	// Create profile
	p := &Profile{
		Name:       "exists-test",
		GitName:    "Test",
		GitEmail:   "test@test.com",
		AuthMethod: AuthSSH,
		CreatedAt:  time.Now(),
		LastUsed:   time.Now(),
	}
	if err := p.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Should exist now
	exists, err = Exists("exists-test")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false for existing profile")
	}
}

func TestProfileDelete(t *testing.T) {
	setupTestDir(t)

	p := &Profile{
		Name:       "delete-test",
		GitName:    "Test",
		GitEmail:   "test@test.com",
		AuthMethod: AuthSSH,
		CreatedAt:  time.Now(),
		LastUsed:   time.Now(),
	}
	if err := p.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := Delete("delete-test"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	exists, _ := Exists("delete-test")
	if exists {
		t.Error("profile still exists after delete")
	}
}

func TestProfileList(t *testing.T) {
	setupTestDir(t)

	// Empty list
	profiles, err := List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("List() returned %d profiles, want 0", len(profiles))
	}

	// Add some profiles
	names := []string{"charlie", "alice", "bob"}
	for _, name := range names {
		p := &Profile{
			Name:       name,
			GitName:    name,
			GitEmail:   name + "@test.com",
			AuthMethod: AuthSSH,
			CreatedAt:  time.Now(),
			LastUsed:   time.Now(),
		}
		if err := p.Save(); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// List should return sorted
	profiles, err = List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(profiles) != 3 {
		t.Fatalf("List() returned %d profiles, want 3", len(profiles))
	}

	// Verify sorted order
	expected := []string{"alice", "bob", "charlie"}
	for i, p := range profiles {
		if p.Name != expected[i] {
			t.Errorf("profiles[%d].Name = %q, want %q", i, p.Name, expected[i])
		}
	}
}

func TestLoadNonexistentProfile(t *testing.T) {
	setupTestDir(t)

	_, err := Load("does-not-exist")
	if err == nil {
		t.Error("Load() should return error for nonexistent profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Load() error = %q, want error containing 'not found'", err.Error())
	}
}

func TestProfileHasAuth(t *testing.T) {
	tests := []struct {
		method  AuthMethod
		hasSSH  bool
		hasHTTP bool
	}{
		{AuthSSH, true, false},
		{AuthHTTP, false, true},
		{AuthBoth, true, true},
	}

	for _, tt := range tests {
		p := &Profile{AuthMethod: tt.method}
		if p.HasSSH() != tt.hasSSH {
			t.Errorf("AuthMethod=%s: HasSSH() = %v, want %v", tt.method, p.HasSSH(), tt.hasSSH)
		}
		if p.HasHTTP() != tt.hasHTTP {
			t.Errorf("AuthMethod=%s: HasHTTP() = %v, want %v", tt.method, p.HasHTTP(), tt.hasHTTP)
		}
	}
}

func TestProfileAuthDisplay(t *testing.T) {
	tests := []struct {
		method  AuthMethod
		display string
	}{
		{AuthSSH, "ssh"},
		{AuthHTTP, "http"},
		{AuthBoth, "ssh+http"},
	}

	for _, tt := range tests {
		p := &Profile{AuthMethod: tt.method}
		if p.AuthDisplay() != tt.display {
			t.Errorf("AuthMethod=%s: AuthDisplay() = %q, want %q", tt.method, p.AuthDisplay(), tt.display)
		}
	}
}

func TestTouchLastUsed(t *testing.T) {
	setupTestDir(t)

	p := &Profile{
		Name:       "touch-test",
		GitName:    "Test",
		GitEmail:   "test@test.com",
		AuthMethod: AuthSSH,
		CreatedAt:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		LastUsed:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	p.Save()

	before := p.LastUsed
	if err := p.TouchLastUsed(); err != nil {
		t.Fatalf("TouchLastUsed() error = %v", err)
	}

	if !p.LastUsed.After(before) {
		t.Error("TouchLastUsed() did not update timestamp")
	}

	// Verify persisted
	loaded, _ := Load("touch-test")
	if loaded.LastUsed.Equal(before) {
		t.Error("TouchLastUsed() did not persist to disk")
	}
}

func TestDeleteNonexistent(t *testing.T) {
	setupTestDir(t)
	// Should not error
	if err := Delete("nonexistent"); err != nil {
		t.Errorf("Delete(nonexistent) error = %v", err)
	}
}

func TestValidateName(t *testing.T) {
	valid := []string{"personal", "work", "my-client", "abc123", "a", "1start"}
	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", name, err)
		}
	}

	invalid := []string{"", "My-Profile", "has space", "has.dot", "-starts-dash", "a!b", strings.Repeat("x", 33)}
	for _, name := range invalid {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) = nil, want error", name)
		}
	}
}

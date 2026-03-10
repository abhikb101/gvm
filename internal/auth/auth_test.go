package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSSHKeyPath(t *testing.T) {
	path, err := SSHKeyPath("personal")
	if err != nil {
		t.Fatalf("SSHKeyPath() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".ssh", "gvm_personal")
	if path != expected {
		t.Errorf("SSHKeyPath(\"personal\") = %q, want %q", path, expected)
	}
}

func TestSSHKeyPathDifferentProfiles(t *testing.T) {
	profiles := []string{"work", "personal", "client-a"}
	for _, name := range profiles {
		path, err := SSHKeyPath(name)
		if err != nil {
			t.Errorf("SSHKeyPath(%q) error = %v", name, err)
			continue
		}
		if !filepath.IsAbs(path) {
			t.Errorf("SSHKeyPath(%q) = %q, want absolute path", name, path)
		}
		base := filepath.Base(path)
		if base != "gvm_"+name {
			t.Errorf("SSHKeyPath(%q) base = %q, want %q", name, base, "gvm_"+name)
		}
	}
}

func TestSSHKeyExistsNotFound(t *testing.T) {
	exists, err := SSHKeyExists("nonexistent-profile-xyz")
	if err != nil {
		t.Fatalf("SSHKeyExists() error = %v", err)
	}
	if exists {
		t.Error("SSHKeyExists() = true for nonexistent profile")
	}
}

func TestSSHKeyExistsWithKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	os.MkdirAll(sshDir, 0700)

	// Create a fake key
	keyPath := filepath.Join(sshDir, "gvm_test-exists")
	os.WriteFile(keyPath, []byte("fake-key"), 0600)

	exists, err := SSHKeyExists("test-exists")
	if err != nil {
		t.Fatalf("SSHKeyExists() error = %v", err)
	}
	if !exists {
		t.Error("SSHKeyExists() = false for existing key")
	}
}

func TestGenerateSSHKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	os.MkdirAll(sshDir, 0700)

	keyPath, err := GenerateSSHKey("test-gen")
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	// Verify private key exists
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("private key not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("private key permissions = %o, want 0600", info.Mode().Perm())
	}

	// Verify public key exists
	_, err = os.Stat(keyPath + ".pub")
	if err != nil {
		t.Fatalf("public key not created: %v", err)
	}

	// Verify private key content
	data, _ := os.ReadFile(keyPath)
	if len(data) == 0 {
		t.Error("private key is empty")
	}

	// Verify public key has the comment
	pubData, _ := os.ReadFile(keyPath + ".pub")
	if len(pubData) == 0 {
		t.Error("public key is empty")
	}
}

func TestDeleteSSHKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	os.MkdirAll(sshDir, 0700)

	// Create fake keys
	keyPath := filepath.Join(sshDir, "gvm_delete-test")
	os.WriteFile(keyPath, []byte("private"), 0600)
	os.WriteFile(keyPath+".pub", []byte("public"), 0644)

	err := DeleteSSHKey("delete-test")
	if err != nil {
		t.Fatalf("DeleteSSHKey() error = %v", err)
	}

	// Verify both keys deleted
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Error("private key still exists after delete")
	}
	if _, err := os.Stat(keyPath + ".pub"); !os.IsNotExist(err) {
		t.Error("public key still exists after delete")
	}
}

func TestDeleteSSHKeyNonexistent(t *testing.T) {
	// Should not error when key doesn't exist
	err := DeleteSSHKey("definitely-not-a-real-profile")
	if err != nil {
		t.Errorf("DeleteSSHKey() for nonexistent key error = %v", err)
	}
}

func TestRemoveKeyFromAgentNonexistent(t *testing.T) {
	// Should not error for nonexistent key
	err := RemoveKeyFromAgent("/nonexistent/path")
	if err != nil {
		t.Errorf("RemoveKeyFromAgent() error = %v", err)
	}
}

func TestAddKeyToAgent(t *testing.T) {
	// This test verifies AddKeyToAgent doesn't panic
	// The actual ssh-add may fail in CI but shouldn't crash
	home := t.TempDir()
	t.Setenv("HOME", home)

	_ = AddKeyToAgent("/nonexistent/key/path")
	// We just verify it doesn't panic
}

func TestVerifySSHConnectionBadKey(t *testing.T) {
	err := VerifySSHConnection("/nonexistent/key", "nobody")
	if err == nil {
		t.Error("VerifySSHConnection() should error for nonexistent key")
	}
}

func TestVerifyGitHubTokenInvalid(t *testing.T) {
	_, err := VerifyGitHubToken("ghp_clearly_invalid_token")
	if err == nil {
		t.Error("VerifyGitHubToken() should error for invalid token")
	}
}

func TestTokenStatusEmpty(t *testing.T) {
	status := TokenStatus("")
	if status != "not configured" {
		t.Errorf("TokenStatus(\"\") = %q, want \"not configured\"", status)
	}
}

func TestSSHConnectionStatusBadKey(t *testing.T) {
	status := SSHConnectionStatus("/nonexistent", "nobody")
	if status != "failed" {
		t.Errorf("SSHConnectionStatus() = %q, want \"failed\"", status)
	}
}

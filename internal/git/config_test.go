package git

import (
	"os"
	"os/exec"
	"testing"
)

func TestConfigureIdentityLocal(t *testing.T) {
	repoDir := initTestRepo(t)
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoDir)

	err := ConfigureIdentity("local", "Test User", "test@example.com", "/tmp/fake-key")
	if err != nil {
		t.Fatalf("ConfigureIdentity(local) error = %v", err)
	}

	// Verify user.name
	out, _ := exec.Command("git", "config", "--local", "user.name").Output()
	if name := trimOutput(out); name != "Test User" {
		t.Errorf("user.name = %q, want %q", name, "Test User")
	}

	// Verify user.email
	out, _ = exec.Command("git", "config", "--local", "user.email").Output()
	if email := trimOutput(out); email != "test@example.com" {
		t.Errorf("user.email = %q, want %q", email, "test@example.com")
	}

	// Verify core.sshCommand
	out, _ = exec.Command("git", "config", "--local", "core.sshCommand").Output()
	if cmd := trimOutput(out); cmd == "" {
		t.Error("core.sshCommand not set")
	}
}

func TestConfigureIdentityGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Need a repo context for git to work
	repoDir := initTestRepo(t)
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoDir)

	// global config without sshKeyPath
	err := ConfigureIdentity("global", "Global User", "global@example.com", "")
	if err != nil {
		t.Fatalf("ConfigureIdentity(global) error = %v", err)
	}

	out, _ := exec.Command("git", "config", "--global", "user.name").Output()
	if name := trimOutput(out); name != "Global User" {
		t.Errorf("global user.name = %q, want %q", name, "Global User")
	}
}

func TestConfigureIdentityGlobalSSHKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repoDir := initTestRepo(t)
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoDir)

	// Switching to a profile with an SSH key pins it globally
	err := ConfigureIdentity("global", "Work User", "work@example.com", "/tmp/work-key")
	if err != nil {
		t.Fatalf("ConfigureIdentity(global) error = %v", err)
	}

	out, _ := exec.Command("git", "config", "--global", "core.sshCommand").Output()
	if cmd := trimOutput(out); cmd == "" {
		t.Error("global core.sshCommand not set")
	}

	// Switching to a profile without an SSH key clears the GVM-managed value
	err = ConfigureIdentity("global", "Other User", "other@example.com", "")
	if err != nil {
		t.Fatalf("ConfigureIdentity(global, no key) error = %v", err)
	}

	out, _ = exec.Command("git", "config", "--global", "core.sshCommand").Output()
	if cmd := trimOutput(out); cmd != "" {
		t.Errorf("global core.sshCommand = %q, want unset after switching to no-key profile", cmd)
	}
}

func TestConfigureIdentityPreservesUserSSHCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repoDir := initTestRepo(t)
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoDir)

	// A user-managed core.sshCommand (not GVM's format) must not be removed
	userCmd := "ssh -F /home/me/custom-ssh-config"
	if err := SetGlobalConfig("core.sshCommand", userCmd); err != nil {
		t.Fatalf("SetGlobalConfig() error = %v", err)
	}

	if err := ConfigureIdentity("global", "User", "u@example.com", ""); err != nil {
		t.Fatalf("ConfigureIdentity() error = %v", err)
	}

	out, _ := exec.Command("git", "config", "--global", "core.sshCommand").Output()
	if cmd := trimOutput(out); cmd != userCmd {
		t.Errorf("global core.sshCommand = %q, want %q (user config preserved)", cmd, userCmd)
	}
}

func TestSetLocalConfig(t *testing.T) {
	repoDir := initTestRepo(t)
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoDir)

	err := SetLocalConfig("user.name", "LocalTest")
	if err != nil {
		t.Fatalf("SetLocalConfig() error = %v", err)
	}

	val, err := GetLocalConfig("user.name")
	if err != nil {
		t.Fatalf("GetLocalConfig() error = %v", err)
	}
	if val != "LocalTest" {
		t.Errorf("GetLocalConfig() = %q, want %q", val, "LocalTest")
	}
}

func TestUnsetLocalConfig(t *testing.T) {
	repoDir := initTestRepo(t)
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoDir)

	_ = SetLocalConfig("test.key", "value")
	_ = UnsetLocalConfig("test.key")

	_, err := GetLocalConfig("test.key")
	if err == nil {
		t.Error("key should be unset after UnsetLocalConfig")
	}
}

func TestConfigureCredentialHelper(t *testing.T) {
	repoDir := initTestRepo(t)
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoDir)

	err := ConfigureCredentialHelper("local", "test-profile")
	if err != nil {
		t.Fatalf("ConfigureCredentialHelper() error = %v", err)
	}

	val, err := GetLocalConfig("credential.https://github.com.helper")
	if err != nil {
		t.Fatalf("reading credential helper: %v", err)
	}
	if val == "" {
		t.Error("credential helper not set")
	}
}

func trimOutput(b []byte) string {
	s := string(b)
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return s
}

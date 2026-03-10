package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallHook(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Install zsh hook
	installed, err := InstallHook(Zsh)
	if err != nil {
		t.Fatalf("InstallHook(zsh) error = %v", err)
	}
	if !installed {
		t.Error("InstallHook(zsh) = false, want true")
	}

	// Verify content
	content, _ := os.ReadFile(filepath.Join(home, ".zshrc"))
	if !strings.Contains(string(content), hookMarkerStart) {
		t.Error("hook marker not found in .zshrc")
	}
	if !strings.Contains(string(content), "gvm_auto_switch") {
		t.Error("gvm_auto_switch function not found in .zshrc")
	}

	// Idempotent — second install should return false
	installed, err = InstallHook(Zsh)
	if err != nil {
		t.Fatalf("second InstallHook(zsh) error = %v", err)
	}
	if installed {
		t.Error("second InstallHook(zsh) = true, want false (already installed)")
	}
}

func TestIsHookInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if IsHookInstalled(Zsh) {
		t.Error("IsHookInstalled() = true before install")
	}

	InstallHook(Zsh)

	if !IsHookInstalled(Zsh) {
		t.Error("IsHookInstalled() = false after install")
	}
}

func TestUninstallHook(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Write some existing content
	rcPath := filepath.Join(home, ".zshrc")
	os.WriteFile(rcPath, []byte("# existing config\nexport PATH=/usr/local/bin:$PATH\n"), 0644)

	// Install
	InstallHook(Zsh)

	// Verify installed
	if !IsHookInstalled(Zsh) {
		t.Fatal("hook not installed")
	}

	// Uninstall
	removed, err := UninstallHook(Zsh)
	if err != nil {
		t.Fatalf("UninstallHook() error = %v", err)
	}
	if !removed {
		t.Error("UninstallHook() = false, want true")
	}

	// Verify removed
	if IsHookInstalled(Zsh) {
		t.Error("IsHookInstalled() = true after uninstall")
	}

	// Verify existing content is preserved
	content, _ := os.ReadFile(rcPath)
	if !strings.Contains(string(content), "export PATH") {
		t.Error("existing config was lost during uninstall")
	}

	// Uninstall again — should be idempotent
	removed, err = UninstallHook(Zsh)
	if err != nil {
		t.Fatalf("second UninstallHook() error = %v", err)
	}
	if removed {
		t.Error("second UninstallHook() = true, want false")
	}
}

func TestInstallHookBash(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	installed, err := InstallHook(Bash)
	if err != nil {
		t.Fatalf("InstallHook(bash) error = %v", err)
	}
	if !installed {
		t.Error("InstallHook(bash) = false")
	}

	content, _ := os.ReadFile(filepath.Join(home, ".bashrc"))
	if !strings.Contains(string(content), "PROMPT_COMMAND") {
		t.Error("bash hook missing PROMPT_COMMAND")
	}
}

func TestInstallHookFish(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	installed, err := InstallHook(Fish)
	if err != nil {
		t.Fatalf("InstallHook(fish) error = %v", err)
	}
	if !installed {
		t.Error("InstallHook(fish) = false")
	}

	content, _ := os.ReadFile(filepath.Join(home, ".config", "fish", "config.fish"))
	if !strings.Contains(string(content), "--on-variable PWD") {
		t.Error("fish hook missing --on-variable PWD")
	}
}

func TestHookScriptContent(t *testing.T) {
	// Verify all hooks contain the essential functions
	for _, s := range []Shell{Zsh, Bash, Fish} {
		script := HookScript(s)
		if !strings.Contains(script, "gvm_auto_switch") {
			t.Errorf("%s hook missing gvm_auto_switch", s)
		}
		if !strings.Contains(script, "gvm_prompt_info") {
			t.Errorf("%s hook missing gvm_prompt_info", s)
		}
		if !strings.Contains(script, ".gvmrc") {
			t.Errorf("%s hook missing .gvmrc reference", s)
		}
	}
}

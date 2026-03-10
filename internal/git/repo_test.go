package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", dir)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"HOME="+dir,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s\n%s", err, out)
	}
	return dir
}

func TestFindRepoRoot(t *testing.T) {
	repoDir := initTestRepo(t)

	// From repo root
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()

	_ = os.Chdir(repoDir)
	root, err := FindRepoRoot()
	if err != nil {
		t.Fatalf("FindRepoRoot() error = %v", err)
	}

	// Normalize paths for comparison
	repoDir, _ = filepath.EvalSymlinks(repoDir)
	root, _ = filepath.EvalSymlinks(root)
	if root != repoDir {
		t.Errorf("FindRepoRoot() = %q, want %q", root, repoDir)
	}

	// From subdirectory
	subDir := filepath.Join(repoDir, "src", "pkg")
	_ = os.MkdirAll(subDir, 0755)
	_ = os.Chdir(subDir)

	root, err = FindRepoRoot()
	if err != nil {
		t.Fatalf("FindRepoRoot() from subdir error = %v", err)
	}
	root, _ = filepath.EvalSymlinks(root)
	if root != repoDir {
		t.Errorf("FindRepoRoot() from subdir = %q, want %q", root, repoDir)
	}
}

func TestFindRepoRootNotARepo(t *testing.T) {
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()

	_ = os.Chdir(dir)
	_, err := FindRepoRoot()
	if err == nil {
		t.Error("FindRepoRoot() should return error outside a repo")
	}
}

func TestIsInsideRepo(t *testing.T) {
	repoDir := initTestRepo(t)
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()

	_ = os.Chdir(repoDir)
	if !IsInsideRepo() {
		t.Error("IsInsideRepo() = false inside a repo")
	}

	notRepo := t.TempDir()
	_ = os.Chdir(notRepo)
	if IsInsideRepo() {
		t.Error("IsInsideRepo() = true outside a repo")
	}
}

func TestWriteReadGVMRC(t *testing.T) {
	dir := t.TempDir()

	// Write
	if err := WriteGVMRC(dir, "personal"); err != nil {
		t.Fatalf("WriteGVMRC() error = %v", err)
	}

	// Read
	name, err := ReadGVMRC(dir)
	if err != nil {
		t.Fatalf("ReadGVMRC() error = %v", err)
	}
	if name != "personal" {
		t.Errorf("ReadGVMRC() = %q, want %q", name, "personal")
	}

	// Overwrite
	if err := WriteGVMRC(dir, "work"); err != nil {
		t.Fatalf("WriteGVMRC() overwrite error = %v", err)
	}

	name, err = ReadGVMRC(dir)
	if err != nil {
		t.Fatalf("ReadGVMRC() error = %v", err)
	}
	if name != "work" {
		t.Errorf("ReadGVMRC() after overwrite = %q, want %q", name, "work")
	}
}

func TestReadGVMRCNotExists(t *testing.T) {
	dir := t.TempDir()

	name, err := ReadGVMRC(dir)
	if err != nil {
		t.Fatalf("ReadGVMRC() error = %v", err)
	}
	if name != "" {
		t.Errorf("ReadGVMRC() = %q, want empty", name)
	}
}

func TestRemoveGVMRC(t *testing.T) {
	dir := t.TempDir()

	_ = WriteGVMRC(dir, "test")

	if err := RemoveGVMRC(dir); err != nil {
		t.Fatalf("RemoveGVMRC() error = %v", err)
	}

	name, _ := ReadGVMRC(dir)
	if name != "" {
		t.Errorf("ReadGVMRC() after remove = %q, want empty", name)
	}

	// Removing again should not error
	if err := RemoveGVMRC(dir); err != nil {
		t.Errorf("RemoveGVMRC() on non-existent should not error: %v", err)
	}
}

func TestEnsureGlobalGitignore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := EnsureGlobalGitignore(); err != nil {
		t.Fatalf("EnsureGlobalGitignore() error = %v", err)
	}

	// Verify file content
	ignorePath := filepath.Join(home, ".config", "git", "ignore")
	content, err := os.ReadFile(ignorePath)
	if err != nil {
		t.Fatalf("reading gitignore: %v", err)
	}
	if !strings.Contains(string(content), ".gvmrc") {
		t.Error("global gitignore does not contain .gvmrc")
	}

	// Idempotent — calling again should not duplicate
	if err := EnsureGlobalGitignore(); err != nil {
		t.Fatalf("second EnsureGlobalGitignore() error = %v", err)
	}

	content, _ = os.ReadFile(ignorePath)
	count := strings.Count(string(content), ".gvmrc")
	if count != 1 {
		t.Errorf("global gitignore contains .gvmrc %d times, want 1", count)
	}
}

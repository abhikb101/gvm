package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gvm-tools/gvm/internal/fsutil"
)

const gvmrcFile = ".gvmrc"

// IsInsideRepo returns true if the current directory is inside a git repository.
func IsInsideRepo() bool {
	_, err := FindRepoRoot()
	return err == nil
}

// FindRepoRoot walks up from the current directory to find the nearest .git directory.
// Returns the repository root path.
func FindRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current directory: %w", err)
	}

	for {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			if info.IsDir() {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not a git repository (or any parent up to /)")
}

// FindGVMRC walks up from the current directory to find the nearest .gvmrc file.
// Returns the path to the .gvmrc and the profile name it contains.
func FindGVMRC() (string, string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("getting current directory: %w", err)
	}

	home, _ := os.UserHomeDir()

	for {
		gvmrcPath := filepath.Join(dir, gvmrcFile)
		if _, err := os.Stat(gvmrcPath); err == nil {
			data, err := os.ReadFile(gvmrcPath)
			if err != nil {
				return "", "", fmt.Errorf("reading %s: %w", gvmrcPath, err)
			}
			profileName := strings.TrimSpace(string(data))
			if profileName != "" {
				return gvmrcPath, profileName, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir || dir == home {
			break
		}
		dir = parent
	}

	return "", "", fmt.Errorf("no .gvmrc found")
}

// WriteGVMRC creates or updates the .gvmrc file at the repo root.
func WriteGVMRC(repoRoot, profileName string) error {
	path := filepath.Join(repoRoot, gvmrcFile)
	return fsutil.AtomicWrite(path, []byte(profileName+"\n"), 0644)
}

// ReadGVMRC reads the profile name from a .gvmrc file at the given repo root.
func ReadGVMRC(repoRoot string) (string, error) {
	path := filepath.Join(repoRoot, gvmrcFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("reading .gvmrc: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// RemoveGVMRC deletes the .gvmrc file from a repo root.
func RemoveGVMRC(repoRoot string) error {
	path := filepath.Join(repoRoot, gvmrcFile)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing .gvmrc: %w", err)
	}
	return nil
}

// EnsureGlobalGitignore ensures .gvmrc is listed in the global gitignore.
func EnsureGlobalGitignore() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("determining home directory: %w", err)
	}

	// Use ~/.config/git/ignore as per git convention
	ignoreDir := filepath.Join(home, ".config", "git")
	if err := os.MkdirAll(ignoreDir, 0755); err != nil {
		return fmt.Errorf("creating git config directory: %w", err)
	}

	ignorePath := filepath.Join(ignoreDir, "ignore")

	// Read existing content
	content, err := os.ReadFile(ignorePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading global gitignore: %w", err)
	}

	// Check if .gvmrc is already ignored
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == ".gvmrc" {
			return nil // already present
		}
	}

	// Append .gvmrc
	f, err := os.OpenFile(ignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening global gitignore: %w", err)
	}
	defer f.Close()

	entry := ".gvmrc\n"
	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		entry = "\n" + entry
	}

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("writing to global gitignore: %w", err)
	}

	return nil
}

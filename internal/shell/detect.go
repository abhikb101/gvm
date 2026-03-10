package shell

import (
	"os"
	"path/filepath"
	"strings"
)

// Shell represents a supported shell type.
type Shell string

const (
	Zsh  Shell = "zsh"
	Bash Shell = "bash"
	Fish Shell = "fish"
)

// Detect returns the current user's shell.
func Detect() Shell {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return Bash // safe default
	}

	base := filepath.Base(shellPath)
	switch {
	case strings.Contains(base, "zsh"):
		return Zsh
	case strings.Contains(base, "fish"):
		return Fish
	default:
		return Bash
	}
}

// ConfigFile returns the shell's RC file path.
func (s Shell) ConfigFile() string {
	home, _ := os.UserHomeDir()

	switch s {
	case Zsh:
		return filepath.Join(home, ".zshrc")
	case Fish:
		return filepath.Join(home, ".config", "fish", "config.fish")
	default:
		return filepath.Join(home, ".bashrc")
	}
}

// String returns the shell name.
func (s Shell) String() string {
	return string(s)
}

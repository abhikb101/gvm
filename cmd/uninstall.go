package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gvm-tools/gvm/internal/auth"
	"github.com/gvm-tools/gvm/internal/config"
	"github.com/gvm-tools/gvm/internal/fsutil"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/shell"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var uninstallKeepKeys bool

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove all GVM data and reverse shell changes",
	Long: `Cleanly remove GVM from your system:
  - Remove shell hook from your RC file
  - Delete all GVM-managed SSH keys (unless --keep-keys)
  - Remove ~/.gvm/ config directory
  - Remove .gvmrc from global gitignore

This does NOT uninstall the gvm binary itself.`,
	RunE: runUninstall,
}

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallKeepKeys, "keep-keys", false, "do not delete SSH keys")
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(ui.Bold("GVM Uninstall"))
	fmt.Println()
	fmt.Println("This will remove all GVM configuration and data:")
	fmt.Println("  - Shell hook from your RC file")
	if !uninstallKeepKeys {
		fmt.Println("  - All GVM-managed SSH keys (~/.ssh/gvm_*)")
	}
	fmt.Println("  - Config directory (~/.gvm/)")
	fmt.Println("  - .gvmrc entry from global gitignore")
	fmt.Println()

	if !promptConfirm(reader, "Are you sure? This cannot be undone. (y/N): ") {
		fmt.Println("Cancelled.")
		return nil
	}

	// 1. Remove shell hook
	for _, s := range []shell.Shell{shell.Zsh, shell.Bash, shell.Fish} {
		removed, err := shell.UninstallHook(s)
		if err != nil {
			ui.Warn("Could not remove %s hook: %v", s, err)
		} else if removed {
			ui.Success("Removed shell hook from %s", s.ConfigFile())
		}
	}

	// 2. Delete SSH keys for all profiles
	if !uninstallKeepKeys {
		profiles, err := profile.List()
		if err == nil {
			for _, p := range profiles {
				if p.SSHKeyPath != "" {
					if err := auth.DeleteSSHKey(p.Name); err != nil {
						ui.Warn("Could not delete SSH key for '%s': %v", p.Name, err)
					} else {
						ui.Success("Deleted SSH key for '%s'", p.Name)
					}
				}
				// Also delete keychain entries
				deleteToken(p)
			}
		}
	}

	// 3. Remove .gvmrc from global gitignore
	removeFromGlobalGitignore()

	// 4. Delete ~/.gvm/ directory
	dir, err := config.Dir()
	if err == nil {
		if err := os.RemoveAll(dir); err != nil {
			ui.Warn("Could not delete %s: %v", dir, err)
		} else {
			ui.Success("Deleted %s", dir)
		}
	}

	fmt.Println()
	ui.Success("GVM has been uninstalled")
	fmt.Println("  The 'gvm' binary is still at its install location.")
	fmt.Println("  Remove it manually, e.g.: rm $(which gvm)")
	fmt.Println("  Or: brew uninstall gvm")

	return nil
}

func removeFromGlobalGitignore() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	for _, path := range []string{
		filepath.Join(home, ".config", "git", "ignore"),
		filepath.Join(home, ".gitignore_global"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := splitLines(string(data))
		var newLines []string
		changed := false
		for _, line := range lines {
			if line == ".gvmrc" {
				changed = true
				continue
			}
			newLines = append(newLines, line)
		}

		if changed {
			content := ""
			for i, line := range newLines {
				content += line
				if i < len(newLines)-1 {
					content += "\n"
				}
			}
			if len(newLines) > 0 {
				content += "\n"
			}
			if err := fsutil.AtomicWrite(path, []byte(content), 0644); err != nil {
				ui.Warn("Could not update %s: %v", path, err)
			} else {
				ui.Success("Removed .gvmrc from %s", path)
			}
		}
	}
}

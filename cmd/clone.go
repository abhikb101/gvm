package cmd

import (
	"fmt"
	"os"

	"github.com/gvm-tools/gvm/internal/config"
	gitpkg "github.com/gvm-tools/gvm/internal/git"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone <profile-name> <repo-url> [directory]",
	Short: "Clone a repository with the correct identity",
	Long:  "Clone a git repository using the specified profile's credentials and automatically bind it.",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runClone,
}

func init() {
	rootCmd.AddCommand(cloneCmd)
}

func runClone(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		return fmt.Errorf("GVM not initialized — run 'gvm init' first")
	}

	name := args[0]
	repoURL := args[1]
	var targetDir string
	if len(args) > 2 {
		targetDir = args[2]
	}

	p, err := profile.Load(name)
	if err != nil {
		return err
	}

	// Warn about auth mismatch
	if warning := gitpkg.DetectURLAuthMismatch(repoURL, p); warning != "" {
		ui.Warn("%s", warning)
	}

	ui.Info("Cloning with identity '%s'...", name)

	clonedPath, err := gitpkg.CloneWithProfile(p, repoURL, targetDir)
	if err != nil {
		return err
	}

	// cd into the cloned repo and run the equivalent of `gvm use`
	if err := gitpkg.WriteGVMRC(clonedPath, name); err != nil {
		ui.Warn("Could not create .gvmrc: %v", err)
	}

	// Set local git config in the cloned repo
	origDir, _ := os.Getwd()
	if err := os.Chdir(clonedPath); err == nil {
		if err := gitpkg.ConfigureIdentity("local", p.GitName, p.GitEmail, p.SSHKeyPath); err != nil {
			ui.Warn("Could not set local git config: %v", err)
		}
		if p.HasHTTP() {
			if err := gitpkg.ConfigureCredentialHelper("local", p.Name); err != nil {
				ui.Warn("Could not configure credential helper: %v", err)
			}
		}
		_ = os.Chdir(origDir)
	}

	ui.Success("Bound '%s' to %s", name, clonedPath)
	return nil
}

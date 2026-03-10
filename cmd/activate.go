package cmd

import (
	"fmt"

	"github.com/gvm-tools/gvm/internal/auth"
	"github.com/gvm-tools/gvm/internal/config"
	gitpkg "github.com/gvm-tools/gvm/internal/git"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var activateCmd = &cobra.Command{
	Use:    "_activate <profile-name>",
	Short:  "Internal: activate a profile",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE:   runActivate,
}

var quietFlag bool

func init() {
	activateCmd.Flags().BoolVar(&quietFlag, "quiet", false, "suppress output")
	rootCmd.AddCommand(activateCmd)
}

func runActivate(cmd *cobra.Command, args []string) error {
	return activateProfile(args[0], quietFlag)
}

// activateProfile is the core activation logic used by use, switch, and the shell hook.
func activateProfile(name string, quiet bool) error {
	p, err := profile.Load(name)
	if err != nil {
		// Edge case: .gvmrc references a deleted profile
		exists, _ := profile.Exists(name)
		if !exists && !quiet {
			ui.Warn("This repo is bound to profile '%s' which no longer exists", name)
			ui.Info("Run 'gvm use <new-profile>' to rebind")
		}
		return err
	}

	// Deactivate current profile's SSH key
	current, _ := config.GetActive()
	if current != "" && current != name {
		deactivateProfile(current)
	}

	// Set as active
	if err := config.SetActive(name); err != nil {
		return fmt.Errorf("setting active profile: %w", err)
	}

	// Configure SSH if available
	if p.HasSSH() && p.SSHKeyPath != "" {
		if err := auth.AddKeyToAgent(p.SSHKeyPath); err != nil && !quiet {
			ui.Warn("Could not add SSH key to agent: %v", err)
		}
	}

	// Configure git identity based on context
	inBoundRepo := false
	if gitpkg.IsInsideRepo() {
		repoRoot, err := gitpkg.FindRepoRoot()
		if err == nil {
			gvmrcProfile, _ := gitpkg.ReadGVMRC(repoRoot)
			if gvmrcProfile == name {
				inBoundRepo = true
				if err := gitpkg.ConfigureIdentity("local", p.GitName, p.GitEmail, p.SSHKeyPath); err != nil && !quiet {
					ui.Warn("Could not set local git config: %v", err)
				}
				if p.HasHTTP() {
					gitpkg.ConfigureCredentialHelper("local", p.Name)
				}
			}
		}
	}

	// If not in a bound repo, set global git config
	if !inBoundRepo {
		if err := gitpkg.ConfigureIdentity("global", p.GitName, p.GitEmail, ""); err != nil && !quiet {
			ui.Warn("Could not set global git config: %v", err)
		}
	}

	// Update last_used
	p.TouchLastUsed()

	if !quiet {
		ui.Success("Active identity: %s <%s>", p.GitName, p.GitEmail)
	}

	return nil
}

package cmd

import (
	"fmt"

	"github.com/gvm-tools/gvm/internal/config"
	gitpkg "github.com/gvm-tools/gvm/internal/git"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch <profile-name>",
	Short: "Switch active identity globally (session-level)",
	Long: `Switch the active profile for the current shell session.
This does NOT bind any repo — it's a temporary session-wide change.
Use 'gvm use' to permanently bind a repo to a profile.`,
	Args: cobra.ExactArgs(1),
	RunE: runSwitch,
}

func init() {
	rootCmd.AddCommand(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		return fmt.Errorf("GVM not initialized — run 'gvm init' first")
	}

	name := args[0]

	p, err := profile.Load(name)
	if err != nil {
		return err
	}

	// Set global git config, pinning the profile's SSH key so remote
	// operations authenticate as this profile too
	if err := gitpkg.ConfigureIdentity("global", p.GitName, p.GitEmail, p.SSHKeyPath); err != nil {
		return fmt.Errorf("setting global git config: %w", err)
	}

	// Activate the profile
	if err := activateProfile(name, true); err != nil {
		return err
	}

	ui.Success("Switched to '%s' globally", name)
	fmt.Printf("  Active identity: %s <%s>\n", p.GitName, p.GitEmail)

	// A repo binding (.gvmrc + local git config) takes precedence over a
	// global switch — tell the user instead of silently using the bound profile
	if gitpkg.IsInsideRepo() {
		if repoRoot, rerr := gitpkg.FindRepoRoot(); rerr == nil {
			if bound, _ := gitpkg.ReadGVMRC(repoRoot); bound != "" && bound != name {
				ui.Warn("This repo is bound to '%s' via .gvmrc, which takes precedence here", bound)
				ui.Info("Run 'gvm use %s' to rebind this repo, or 'gvm unbind' to remove the binding", name)
			}
		}
	}

	return nil
}

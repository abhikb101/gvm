package cmd

import (
	"fmt"

	"github.com/gvm-tools/gvm/internal/config"
	gitpkg "github.com/gvm-tools/gvm/internal/git"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use <profile-name>",
	Short: "Bind the current repository to a profile",
	Long: `Permanently bind the current git repository to a profile.
The binding is stored in a .gvmrc file at the repo root.
When you cd into this repo, GVM auto-switches to this profile.`,
	Args: cobra.ExactArgs(1),
	RunE: runUse,
}

func init() {
	rootCmd.AddCommand(useCmd)
}

func runUse(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		return fmt.Errorf("GVM not initialized — run 'gvm init' first")
	}

	name := args[0]

	// Verify profile exists
	p, err := profile.Load(name)
	if err != nil {
		// Check if any profiles exist at all
		profiles, _ := profile.List()
		if len(profiles) == 0 {
			return fmt.Errorf("no profiles found — run 'gvm init' to get started")
		}
		return err
	}

	// Must be in a git repo
	repoRoot, err := gitpkg.FindRepoRoot()
	if err != nil {
		return fmt.Errorf("not a git repository — use 'gvm switch %s' to change identity globally, or cd into a git repo first", name)
	}

	// Guard: warn if .gvmrc isn't in global gitignore
	if !isGVMRCInGlobalGitignore() {
		ui.Warn(".gvmrc is not in your global gitignore — it may get committed accidentally")
		ui.Info("Run 'gvm init' or add '.gvmrc' to ~/.config/git/ignore")
	}

	// Write .gvmrc
	if err := gitpkg.WriteGVMRC(repoRoot, name); err != nil {
		return fmt.Errorf("writing .gvmrc: %w", err)
	}

	// Set local git config
	if err := gitpkg.ConfigureIdentity("local", p.GitName, p.GitEmail, p.SSHKeyPath); err != nil {
		return fmt.Errorf("configuring git identity: %w", err)
	}

	// Set up credential helper if profile has HTTP auth
	if p.HasHTTP() {
		if err := gitpkg.ConfigureCredentialHelper("local", p.Name); err != nil {
			ui.Warn("Could not configure credential helper: %v", err)
		}
	}

	// Activate this profile
	if err := activateProfile(name, true); err != nil {
		ui.Warn("Could not activate profile: %v", err)
	}

	ui.Success("Bound '%s' to %s", name, repoRoot)
	fmt.Printf("  Active identity: %s <%s>\n", p.GitName, p.GitEmail)

	return nil
}

package cmd

import (
	"fmt"

	"github.com/gvm-tools/gvm/internal/config"
	gitpkg "github.com/gvm-tools/gvm/internal/git"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var unbindCmd = &cobra.Command{
	Use:   "unbind",
	Short: "Remove profile binding from the current repository",
	Long:  "Remove the .gvmrc file and local git identity config from the current repo.",
	RunE:  runUnbind,
}

func init() {
	rootCmd.AddCommand(unbindCmd)
}

func runUnbind(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		return fmt.Errorf("GVM not initialized — run 'gvm init' first")
	}

	repoRoot, err := gitpkg.FindRepoRoot()
	if err != nil {
		return fmt.Errorf("not inside a git repository")
	}

	current, _ := gitpkg.ReadGVMRC(repoRoot)
	if current == "" {
		fmt.Println("No profile is bound to this repository.")
		return nil
	}

	if err := gitpkg.RemoveGVMRC(repoRoot); err != nil {
		return fmt.Errorf("removing .gvmrc: %w", err)
	}

	// Clean up local git config set by GVM
	_ = gitpkg.UnsetLocalConfig("core.sshCommand")
	_ = gitpkg.UnsetLocalConfig("credential.https://github.com.helper")

	ui.Success("Unbound profile '%s' from %s", current, repoRoot)
	return nil
}

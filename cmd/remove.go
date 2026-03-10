package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gvm-tools/gvm/internal/auth"
	"github.com/gvm-tools/gvm/internal/config"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var (
	removeForce bool
)

var removeCmd = &cobra.Command{
	Use:     "remove <profile-name>",
	Aliases: []string{"rm"},
	Short:   "Delete a profile and its associated credentials",
	Args:    cobra.ExactArgs(1),
	RunE:    runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "skip confirmation prompt")
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		return fmt.Errorf("GVM not initialized — run 'gvm init' first")
	}

	name := args[0]

	p, err := profile.Load(name)
	if err != nil {
		return err
	}

	// Confirm unless --force
	if !removeForce {
		fmt.Printf("Remove profile '%s'? This will:\n", name)
		if p.SSHKeyPath != "" {
			fmt.Printf("  - Delete SSH key %s\n", p.SSHKeyPath)
		}
		if p.GHTokenEncrypted != "" {
			fmt.Println("  - Revoke GitHub OAuth token")
		}
		fmt.Println("  - Remove all repo bindings for this profile")
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Confirm (y/N): ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Delete SSH key
	if p.HasSSH() {
		if err := auth.DeleteSSHKey(name); err != nil {
			ui.Warn("Could not delete SSH key: %v", err)
		}
	}

	// Revoke and delete stored token
	if p.GHTokenEncrypted != "" {
		token, err := loadToken(p)
		if err == nil && token != "" {
			cfg, cfgErr := config.Load()
			if cfgErr == nil {
				auth.RevokeToken(cfg.GetGitHubClientID(), token)
			}
		}
		deleteToken(p)
	}

	// Delete profile file
	if err := profile.Delete(name); err != nil {
		return fmt.Errorf("deleting profile: %w", err)
	}

	// If this was the active profile, clear active state
	active, _ := config.GetActive()
	if active == name {
		_ = config.ClearActive()
	}

	ui.Success("Profile '%s' removed", name)
	ui.Info("Any repos bound to '%s' via .gvmrc will need to be rebound — run 'gvm use <new-profile>' in those repos", name)
	return nil
}

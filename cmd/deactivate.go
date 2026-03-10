package cmd

import (
	"github.com/gvm-tools/gvm/internal/auth"
	"github.com/gvm-tools/gvm/internal/config"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var deactivateQuiet bool

var deactivateCmd = &cobra.Command{
	Use:    "_deactivate",
	Short:  "Internal: deactivate current profile",
	Hidden: true,
	RunE:   runDeactivate,
}

func init() {
	deactivateCmd.Flags().BoolVar(&deactivateQuiet, "quiet", false, "suppress output")
	rootCmd.AddCommand(deactivateCmd)
}

func runDeactivate(cmd *cobra.Command, args []string) error {
	current, err := config.GetActive()
	if err != nil || current == "" {
		return nil
	}

	deactivateProfile(current)

	if err := config.ClearActive(); err != nil {
		return err
	}

	if !deactivateQuiet {
		ui.Success("Deactivated profile '%s'", current)
	}
	return nil
}

// deactivateProfile removes the SSH key from the agent for the given profile.
func deactivateProfile(name string) {
	p, err := profile.Load(name)
	if err != nil {
		return
	}
	if p.SSHKeyPath != "" {
		_ = auth.RemoveKeyFromAgent(p.SSHKeyPath)
	}
}

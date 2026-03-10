package cmd

import (
	"fmt"
	"os"

	"github.com/gvm-tools/gvm/internal/auth"
	"github.com/gvm-tools/gvm/internal/config"
	"github.com/gvm-tools/gvm/internal/crypto"
	"github.com/gvm-tools/gvm/internal/platform"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "Show all profiles and their status",
	RunE:    runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		return fmt.Errorf("GVM not initialized — run 'gvm init' first")
	}

	profiles, err := profile.List()
	if err != nil {
		return err
	}

	if len(profiles) == 0 {
		fmt.Println("No profiles found. Run 'gvm add <name>' to create one.")
		return nil
	}

	active, _ := config.GetActive()

	table := ui.NewTable()
	for _, p := range profiles {
		marker := "  "
		name := p.Name
		if p.Name == active {
			marker = ui.Green("* ")
			name = ui.Bold(p.Name)
		}

		status := profileConnectionStatus(p)

		table.AddRow(
			marker+name,
			p.GitEmail,
			p.AuthDisplay(),
			status,
		)
	}

	table.Render()
	fmt.Println()
	fmt.Println(ui.Dim("* = active in current context"))

	return nil
}

// profileConnectionStatus checks auth health and returns a status string.
func profileConnectionStatus(p *profile.Profile) string {
	if p.HasSSH() {
		if _, err := os.Stat(p.SSHKeyPath); err != nil {
			return ui.Red("✗ key missing")
		}
		if err := auth.VerifySSHConnection(p.SSHKeyPath, p.GitHubUsername); err != nil {
			return ui.Yellow("✗ ssh failed")
		}
	}

	if p.HasHTTP() {
		if p.GHTokenEncrypted == "" {
			return ui.Dim("not configured")
		}
		token, err := decryptProfileToken(p)
		if err != nil || token == "" {
			return ui.Yellow("✗ token error")
		}
		status := auth.TokenStatus(token)
		if status == "expired" {
			return ui.Yellow("✗ expired")
		}
		if status != "connected" {
			return ui.Red("✗ " + status)
		}
	}

	return ui.Green("✓ connected")
}

// decryptProfileToken loads the token from keychain or file encryption.
func decryptProfileToken(p *profile.Profile) (string, error) {
	if p.GHTokenEncrypted == "keychain" {
		return platform.KeychainLoad(p.Name)
	}
	if p.GHTokenEncrypted != "" {
		return crypto.Decrypt(p.GHTokenEncrypted)
	}
	return "", nil
}

package cmd

import (
	"fmt"

	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/spf13/cobra"
)

var credentialHelperCmd = &cobra.Command{
	Use:    "_credential-helper <profile-name>",
	Short:  "Internal: git credential helper",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE:   runCredentialHelper,
}

func init() {
	rootCmd.AddCommand(credentialHelperCmd)
}

func runCredentialHelper(cmd *cobra.Command, args []string) error {
	name := args[0]

	p, err := profile.Load(name)
	if err != nil {
		return err
	}

	token, err := loadToken(p)
	if err != nil {
		return fmt.Errorf("loading token for '%s': %w", name, err)
	}
	if token == "" {
		return fmt.Errorf("no HTTP token stored for profile '%s'", name)
	}

	// Git credential helper protocol
	fmt.Println("protocol=https")
	fmt.Println("host=github.com")
	fmt.Printf("username=%s\n", p.GitHubUsername)
	fmt.Printf("password=%s\n", token)
	fmt.Println()

	return nil
}

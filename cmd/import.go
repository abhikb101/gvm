package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gvm-tools/gvm/internal/config"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import profiles from an exported JSON file",
	Long: `Import profiles from a file previously created by 'gvm export'.
Existing profiles with the same name will NOT be overwritten.
After importing, run 'gvm login <name> ssh' to set up authentication.`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		return fmt.Errorf("GVM not initialized — run 'gvm init' first")
	}

	inputPath := args[0]
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading import file: %w", err)
	}

	var exported exportedData
	if err := json.Unmarshal(data, &exported); err != nil {
		return fmt.Errorf("parsing import file: %w", err)
	}

	if len(exported.Profiles) == 0 {
		fmt.Println("No profiles found in the import file.")
		return nil
	}

	fmt.Printf("Found %d profiles in %s:\n\n", len(exported.Profiles), inputPath)

	reader := bufio.NewReader(os.Stdin)
	imported := 0
	skipped := 0

	for _, ep := range exported.Profiles {
		exists, _ := profile.Exists(ep.Name)
		if exists {
			ui.Warn("Profile '%s' already exists — skipping", ep.Name)
			skipped++
			continue
		}

		fmt.Printf("  %s  %s <%s>  (%s)\n",
			ui.Bold(ep.Name), ep.GitName, ep.GitEmail, ep.AuthMethod)

		if !promptConfirmYes(reader, fmt.Sprintf("  Import '%s'? (Y/n): ", ep.Name)) {
			continue
		}

		p := &profile.Profile{
			Name:           ep.Name,
			GitName:        ep.GitName,
			GitEmail:       ep.GitEmail,
			GitHubUsername: ep.GitHubUsername,
			AuthMethod:     profile.AuthMethod(ep.AuthMethod),
			SigningKey:     ep.SigningKey,
			CreatedAt:      timeNow(),
			LastUsed:       timeNow(),
		}

		if err := p.Validate(); err != nil {
			ui.Warn("Invalid profile '%s': %v — skipping", ep.Name, err)
			skipped++
			continue
		}

		if err := p.Save(); err != nil {
			ui.Warn("Could not save profile '%s': %v", ep.Name, err)
			continue
		}

		ui.Success("Imported '%s'", ep.Name)
		imported++
	}

	fmt.Println()
	if imported > 0 {
		ui.Success("Imported %d profiles", imported)
		ui.Info("Run 'gvm login <name> ssh' or 'gvm login <name> http' to set up auth for each profile")
	}
	if skipped > 0 {
		fmt.Printf("%d skipped (already exist or invalid)\n", skipped)
	}

	return nil
}

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gvm-tools/gvm/internal/config"
	"github.com/gvm-tools/gvm/internal/migrate"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Import existing Git/SSH/GitHub identities into GVM",
	Long: `Scan your machine for existing Git identities and import them as GVM profiles.

Detects:
  - Global git config (user.name, user.email)
  - SSH keys in ~/.ssh/
  - Host entries in ~/.ssh/config
  - GitHub CLI (gh) authentication
  - includeIf directory-based git configs`,
	RunE: runMigrate,
}

var migrateDryRun bool

func init() {
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "show what would be imported without making changes")
	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		return fmt.Errorf("GVM not initialized — run 'gvm init' first")
	}

	fmt.Println(ui.Bold("GVM Migrate"))
	fmt.Println("Scanning for existing identities...")
	fmt.Println()

	identities := migrate.ScanAll()

	if len(identities) == 0 {
		fmt.Println("No existing identities found.")
		fmt.Println("  Use 'gvm add <name>' to create a new profile from scratch.")
		return nil
	}

	fmt.Printf("Found %d existing %s:\n\n", len(identities), pluralize(len(identities), "identity", "identities"))

	for i, id := range identities {
		fmt.Printf("  %s %s\n", ui.Bold(fmt.Sprintf("[%d]", i+1)), id.Description)
		printIdentityDetails(id)
		fmt.Println()
	}

	if migrateDryRun {
		fmt.Println(ui.Dim("(dry run — no changes made)"))
		return nil
	}

	reader := bufio.NewReader(os.Stdin)
	imported := 0

	for i, id := range identities {
		fmt.Printf("%s Import as GVM profile? (y/N/q to quit): ",
			ui.Bold(fmt.Sprintf("[%d]", i+1)))

		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer == "q" || answer == "quit" {
			break
		}
		if answer != "y" && answer != "yes" {
			continue
		}

		p, err := importIdentity(reader, id)
		if err != nil {
			ui.Warn("Could not import: %v", err)
			continue
		}

		if err := p.Save(); err != nil {
			ui.Warn("Could not save profile: %v", err)
			continue
		}

		ui.Success("Imported as profile '%s'", p.Name)
		imported++
	}

	fmt.Println()
	if imported > 0 {
		ui.Success("Imported %d %s", imported, pluralize(imported, "profile", "profiles"))
		fmt.Println("  Run 'gvm list' to see all profiles")
		fmt.Println("  Run 'gvm use <name>' to bind a repo")
	} else {
		fmt.Println("No profiles imported.")
	}

	return nil
}

func importIdentity(reader *bufio.Reader, id migrate.DetectedIdentity) (*profile.Profile, error) {
	// Suggest a name, let user override
	suggestedName := id.Name
	name := promptDefault(reader, "  Profile name", suggestedName)
	if err := profile.ValidateName(name); err != nil {
		return nil, err
	}

	exists, _ := profile.Exists(name)
	if exists {
		return nil, fmt.Errorf("profile '%s' already exists — choose a different name", name)
	}

	p := &profile.Profile{
		Name: name,
	}

	// Fill in what we have, let user confirm/override the rest
	if id.GitName != "" {
		p.GitName = promptDefault(reader, "  Git name", id.GitName)
	} else {
		p.GitName = prompt(reader, "  Git name (for commits): ")
	}
	if p.GitName == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	if id.GitEmail != "" {
		p.GitEmail = promptDefault(reader, "  Git email", id.GitEmail)
	} else {
		p.GitEmail = prompt(reader, "  Git email: ")
	}

	if id.GitHubUsername != "" {
		p.GitHubUsername = promptDefault(reader, "  GitHub username", id.GitHubUsername)
	} else {
		p.GitHubUsername = prompt(reader, "  GitHub username: ")
	}

	// Determine auth method based on what was found
	if id.SSHKeyPath != "" && id.GHToken != "" {
		p.AuthMethod = profile.AuthBoth
	} else if id.GHToken != "" {
		p.AuthMethod = profile.AuthHTTP
	} else {
		p.AuthMethod = profile.AuthSSH
	}

	// Import SSH key if found
	if id.SSHKeyPath != "" {
		if _, err := os.Stat(id.SSHKeyPath); err == nil {
			p.SSHKeyPath = id.SSHKeyPath
			ui.Success("Using existing SSH key: %s", id.SSHKeyPath)
		}
	}

	// Import token if found from gh CLI
	if id.GHToken != "" {
		if err := storeToken(p, id.GHToken); err != nil {
			ui.Warn("Could not store token: %v", err)
		} else {
			ui.Success("Imported GitHub token")
		}
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	now := timeNow()
	p.CreatedAt = now
	p.LastUsed = now

	return p, nil
}

func printIdentityDetails(id migrate.DetectedIdentity) {
	if id.GitName != "" {
		fmt.Printf("    Name:     %s\n", id.GitName)
	}
	if id.GitEmail != "" {
		fmt.Printf("    Email:    %s\n", id.GitEmail)
	}
	if id.GitHubUsername != "" {
		fmt.Printf("    GitHub:   %s\n", id.GitHubUsername)
	}
	if id.SSHKeyPath != "" {
		fmt.Printf("    SSH Key:  %s\n", id.SSHKeyPath)
	}
	if id.SSHHostAlias != "" {
		fmt.Printf("    SSH Host: %s\n", id.SSHHostAlias)
	}
	fmt.Printf("    Source:   %s\n", ui.Dim(id.Source))
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

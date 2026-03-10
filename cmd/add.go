package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gvm-tools/gvm/internal/config"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <profile-name>",
	Short: "Create a new identity profile",
	Long:  "Interactively create a new profile with git identity and authentication setup.",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		return fmt.Errorf("GVM not initialized — run 'gvm init' first")
	}

	name := args[0]
	if err := profile.ValidateName(name); err != nil {
		return err
	}

	exists, err := profile.Exists(name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("profile '%s' already exists — use 'gvm login %s <ssh|http>' to update auth", name, name)
	}

	reader := bufio.NewReader(os.Stdin)
	p, err := promptForNewProfile(reader, name)
	if err != nil {
		return err
	}

	if err := p.Save(); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	fmt.Println()
	ui.Success("Profile '%s' created", p.Name)
	ui.Info("Use 'gvm use %s' in a repo or 'gvm switch %s' to activate globally", p.Name, p.Name)

	return nil
}

func promptForNewProfile(reader *bufio.Reader, name string) (*profile.Profile, error) {
	p := &profile.Profile{Name: name}

	p.GitName = prompt(reader, "Your name (for git commits): ")
	if p.GitName == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	p.GitEmail = prompt(reader, "Your email: ")

	p.GitHubUsername = prompt(reader, "GitHub username: ")

	authStr := promptDefault(reader, "Auth method (ssh/http/both)", "ssh")
	p.AuthMethod = profile.AuthMethod(authStr)

	if err := p.Validate(); err != nil {
		return nil, err
	}

	// Check for duplicate email across profiles
	existing, _ := profile.List()
	for _, ep := range existing {
		if ep.GitEmail == p.GitEmail {
			ui.Warn("Email '%s' is already used by profile '%s'", p.GitEmail, ep.Name)
		}
	}

	if p.HasSSH() {
		if err := setupSSHForProfile(reader, p); err != nil {
			return nil, err
		}
	}

	if p.HasHTTP() {
		if err := setupHTTPForProfile(p); err != nil {
			return nil, err
		}
	}

	now := timeNow()
	p.CreatedAt = now
	p.LastUsed = now

	return p, nil
}

// prompt prints a message and reads a line of input.
func prompt(reader *bufio.Reader, msg string) string {
	fmt.Print(msg)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// promptDefault prints a message with a default value and reads input.
func promptDefault(reader *bufio.Reader, msg, defaultVal string) string {
	fmt.Printf("%s [%s]: ", msg, defaultVal)
	line, _ := reader.ReadString('\n')
	val := strings.TrimSpace(line)
	if val == "" {
		return defaultVal
	}
	return val
}

// promptConfirm asks a y/N question (default no). Returns true if user answers yes.
func promptConfirm(reader *bufio.Reader, msg string) bool {
	fmt.Print(msg)
	line, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes"
}

// promptConfirmYes asks a Y/n question (default yes). Returns false only if user explicitly says no.
func promptConfirmYes(reader *bufio.Reader, msg string) bool {
	fmt.Print(msg)
	line, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "" || answer == "y" || answer == "yes"
}

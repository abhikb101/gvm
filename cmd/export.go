package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gvm-tools/gvm/internal/config"
	"github.com/gvm-tools/gvm/internal/fsutil"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

// exportedData is the portable format — no secrets, no machine-specific paths.
type exportedData struct {
	Version    string            `json:"version"`
	ExportedAt string            `json:"exported_at"`
	Profiles   []exportedProfile `json:"profiles"`
}

type exportedProfile struct {
	Name           string `json:"name"`
	GitName        string `json:"git_name"`
	GitEmail       string `json:"git_email"`
	GitHubUsername string `json:"github_username"`
	AuthMethod     string `json:"auth_method"`
	SigningKey     string `json:"signing_key,omitempty"`
}

var exportCmd = &cobra.Command{
	Use:   "export [file]",
	Short: "Export profiles to a portable JSON file (no secrets)",
	Long: `Export all profiles to a JSON file for backup or transfer to another machine.
Secrets (SSH keys, OAuth tokens) are NOT exported — only identity metadata.
If no file is specified, prints to stdout.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		return fmt.Errorf("GVM not initialized — run 'gvm init' first")
	}

	profiles, err := profile.List()
	if err != nil {
		return err
	}

	if len(profiles) == 0 {
		return fmt.Errorf("no profiles to export")
	}

	data := exportedData{
		Version:    "1",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
	}

	for _, p := range profiles {
		data.Profiles = append(data.Profiles, exportedProfile{
			Name:           p.Name,
			GitName:        p.GitName,
			GitEmail:       p.GitEmail,
			GitHubUsername: p.GitHubUsername,
			AuthMethod:     string(p.AuthMethod),
			SigningKey:     p.SigningKey,
		})
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding export data: %w", err)
	}
	out = append(out, '\n')

	if len(args) == 0 {
		fmt.Print(string(out))
		return nil
	}

	outputPath := args[0]
	if err := fsutil.AtomicWrite(outputPath, out, 0644); err != nil {
		return fmt.Errorf("writing export file: %w", err)
	}

	ui.Success("Exported %d profiles to %s", len(profiles), outputPath)
	ui.Info("Note: SSH keys and tokens are NOT included — re-run 'gvm login' on the new machine")
	return nil
}

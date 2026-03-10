package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	versionStr = "dev"
	commitStr  = "none"
	dateStr    = "unknown"
)

// SetVersionInfo injects build-time version info.
func SetVersionInfo(version, commit, date string) {
	versionStr = version
	commitStr = commit
	dateStr = date
	rootCmd.Version = version
}

var rootCmd = &cobra.Command{
	Use:   "gvm",
	Short: "nvm for Git identities — manage multiple GitHub accounts",
	Long: `GVM (Git Version Manager) lets you switch between multiple Git/GitHub 
identities with a single command. Create profiles for each account,
bind them to repos, and auto-switch as you navigate your filesystem.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       "dev",
}

// Execute runs the root command and returns an exit code:
// 0 for success, 1 for runtime errors, 2 for usage errors.
func Execute() int {
	rootCmd.SetVersionTemplate(
		fmt.Sprintf("gvm version %s (commit %s, built %s)\n", versionStr, commitStr, dateStr),
	)
	err := rootCmd.Execute()
	if err == nil {
		return 0
	}
	// Print error since SilenceErrors is true
	fmt.Fprintf(os.Stderr, "Error: %s\n", err)

	// Cobra flags/args parsing errors are usage errors → exit 2
	if isUsageError(err) {
		return 2
	}
	return 1
}

func isUsageError(err error) bool {
	msg := err.Error()
	for _, prefix := range []string{
		"unknown command",
		"unknown flag",
		"unknown shorthand flag",
		"accepts ",
		"requires ",
		"invalid argument",
	} {
		if len(msg) >= len(prefix) && msg[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

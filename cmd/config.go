package cmd

import (
	"fmt"

	"github.com/gvm-tools/gvm/internal/config"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config [set <key> <value>]",
	Short: "View or edit global GVM settings",
	Long: `View current settings or change them with 'gvm config set <key> <value>'.

Available keys:
  default-auth      Default auth method for new profiles (ssh/http/both)
  auto-switch       Enable/disable auto-switch on cd (true/false)
  prompt            Show active profile in shell prompt (true/false)
  editor            Preferred editor for config files
  github-client-id  GitHub OAuth App client ID`,
	RunE: runConfig,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

func init() {
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		return fmt.Errorf("GVM not initialized — run 'gvm init' first")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Printf("%-16s %s\n", ui.Bold("Shell:"), cfg.Shell)
	fmt.Printf("%-16s %s\n", ui.Bold("Default auth:"), cfg.DefaultAuth)
	fmt.Printf("%-16s %v\n", ui.Bold("Auto-switch:"), boolStr(cfg.AutoSwitch))
	fmt.Printf("%-16s %v\n", ui.Bold("Prompt display:"), boolStr(cfg.PromptDisplay))

	if cfg.Editor != "" {
		fmt.Printf("%-16s %s\n", ui.Bold("Editor:"), cfg.Editor)
	}
	if cfg.GitHubClientID != "" {
		fmt.Printf("%-16s %s\n", ui.Bold("GitHub App ID:"), cfg.GitHubClientID)
	}

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		return fmt.Errorf("GVM not initialized — run 'gvm init' first")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	key, value := args[0], args[1]
	if err := cfg.Set(key, value); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	ui.Success("%s set to '%s'", key, value)
	return nil
}

func boolStr(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

package cmd

import (
	"fmt"

	"github.com/gvm-tools/gvm/internal/config"
	gitpkg "github.com/gvm-tools/gvm/internal/git"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current active identity with full details",
	RunE:  runWhoami,
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}

func runWhoami(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		return fmt.Errorf("GVM not initialized — run 'gvm init' first")
	}

	// Check for repo-level binding first (takes precedence)
	var boundProfile string
	var repoRoot string
	if gitpkg.IsInsideRepo() {
		repoRoot, _ = gitpkg.FindRepoRoot()
		if repoRoot != "" {
			boundProfile, _ = gitpkg.ReadGVMRC(repoRoot)
		}
	}

	active, _ := config.GetActive()

	// Determine effective profile: repo binding > global active
	effectiveName := active
	viaBinding := false
	if boundProfile != "" {
		effectiveName = boundProfile
		viaBinding = true
	}

	if effectiveName == "" {
		fmt.Println("No active profile.")
		fmt.Println("  Run 'gvm switch <name>' to activate a profile globally.")
		fmt.Println("  Run 'gvm use <name>' in a repo to bind a profile.")
		return nil
	}

	p, err := profile.Load(effectiveName)
	if err != nil {
		// Edge case: .gvmrc references a deleted profile
		if viaBinding {
			ui.Warn("This repo is bound to profile '%s' which no longer exists", effectiveName)
			ui.Info("Run 'gvm use <new-profile>' to rebind")
			return nil
		}
		return err
	}

	fmt.Printf("%s  %s\n", ui.Bold("Profile:"), p.Name)
	fmt.Printf("%s     %s\n", ui.Bold("Name:"), p.GitName)
	fmt.Printf("%s    %s\n", ui.Bold("Email:"), p.GitEmail)
	if p.GitHubUsername != "" {
		fmt.Printf("%s   %s\n", ui.Bold("GitHub:"), p.GitHubUsername)
	}
	fmt.Printf("%s     %s\n", ui.Bold("Auth:"), p.AuthDisplay())
	if p.SSHKeyPath != "" {
		fmt.Printf("%s  %s\n", ui.Bold("SSH Key:"), p.SSHKeyPath)
	}

	// Show binding context
	if viaBinding {
		fmt.Printf("%s %s %s\n", ui.Bold("Bound to:"), repoRoot, ui.Dim("(via .gvmrc)"))
	} else if repoRoot != "" {
		fmt.Printf("%s %s\n", ui.Bold("Context:"), ui.Dim("global switch (no .gvmrc in this repo)"))
	} else {
		fmt.Printf("%s %s\n", ui.Bold("Context:"), ui.Dim("global switch"))
	}

	return nil
}

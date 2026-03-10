package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gvm-tools/gvm/internal/auth"
	"github.com/gvm-tools/gvm/internal/config"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/shell"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Health check for GVM setup",
	Long:  "Verify that GVM is properly configured: config, shell hook, profiles, and auth.",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println(ui.Bold("GVM Doctor"))
	fmt.Println("──────────")

	healthy := 0
	issues := 0

	// Check config directory
	dir, err := config.Dir()
	if err != nil {
		ui.Fail("Cannot determine config directory: %v", err)
		issues++
	} else {
		if _, err := os.Stat(dir); err != nil {
			ui.Fail("Config directory does not exist (~/.gvm/) — run 'gvm init'")
			issues++
		} else {
			ui.Success("Config directory exists (~/.gvm/)")
			healthy++
		}
	}

	// Check config file
	cfg, err := config.Load()
	if err != nil {
		ui.Fail("Config file: %v", err)
		issues++
		fmt.Printf("\n%d healthy, %d issue(s) found.\n", healthy, issues)
		if issues > 0 {
			return fmt.Errorf("health check failed")
		}
		return nil
	}
	healthy++

	// Check shell hook
	detectedShell := shell.Shell(cfg.Shell)
	if shell.IsHookInstalled(detectedShell) {
		ui.Success("Shell hook installed (%s)", cfg.Shell)
		healthy++
	} else {
		ui.Fail("Shell hook not installed — add the GVM hook to %s", detectedShell.ConfigFile())
		ui.Info("  Run 'gvm init' or manually add the shell hook")
		issues++
	}

	// Check global gitignore
	if isGVMRCInGlobalGitignore() {
		ui.Success("Global gitignore includes .gvmrc")
		healthy++
	} else {
		ui.Fail("Global gitignore does not include .gvmrc — run 'gvm init'")
		issues++
	}

	// Check each profile
	profiles, err := profile.List()
	if err != nil {
		ui.Fail("Cannot list profiles: %v", err)
		issues++
	} else if len(profiles) == 0 {
		ui.Warn("No profiles configured — run 'gvm add <name>' to create one")
	} else {
		for _, p := range profiles {
			checkProfileHealth(p, &healthy, &issues)
		}
	}

	fmt.Printf("\n%d healthy, %d issue(s) found.\n", healthy, issues)

	if issues > 0 {
		return fmt.Errorf("health check found issues")
	}
	return nil
}

func checkProfileHealth(p *profile.Profile, healthy, issues *int) {
	profileIssues := []string{}

	// Check SSH auth
	if p.HasSSH() {
		if p.SSHKeyPath == "" {
			profileIssues = append(profileIssues,
				fmt.Sprintf("SSH key path not set — run 'gvm login %s ssh'", p.Name))
		} else if _, err := os.Stat(p.SSHKeyPath); err != nil {
			profileIssues = append(profileIssues,
				fmt.Sprintf("SSH key missing at %s — run 'gvm login %s ssh'", p.SSHKeyPath, p.Name))
		} else {
			if err := auth.VerifySSHConnection(p.SSHKeyPath, p.GitHubUsername); err != nil {
				profileIssues = append(profileIssues,
					fmt.Sprintf("SSH connection failed — %v", err))
			}
		}
	}

	// Check HTTP auth with actual API verification
	if p.HasHTTP() {
		if p.GHTokenEncrypted == "" {
			profileIssues = append(profileIssues,
				fmt.Sprintf("OAuth token not configured — run 'gvm login %s http'", p.Name))
		} else {
			token, err := decryptProfileToken(p)
			if err != nil {
				profileIssues = append(profileIssues,
					fmt.Sprintf("Cannot decrypt OAuth token — run 'gvm login %s http'", p.Name))
			} else if token != "" {
				status := auth.TokenStatus(token)
				switch status {
				case "connected":
					// all good
				case "expired":
					profileIssues = append(profileIssues,
						fmt.Sprintf("OAuth token expired — run 'gvm login %s http'", p.Name))
				default:
					profileIssues = append(profileIssues,
						fmt.Sprintf("OAuth token validation failed (%s) — run 'gvm login %s http'", status, p.Name))
				}
			}
		}
	}

	if len(profileIssues) == 0 {
		ui.Success("Profile '%s': %s <%s>", p.Name, p.GitName, p.GitEmail)
		*healthy++
	} else {
		for _, issue := range profileIssues {
			ui.Fail("Profile '%s': %s", p.Name, issue)
			*issues++
		}
	}
}

func isGVMRCInGlobalGitignore() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	for _, path := range []string{
		filepath.Join(home, ".config", "git", "ignore"),
		filepath.Join(home, ".gitignore_global"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range splitLines(string(data)) {
			if line == ".gvmrc" {
				return true
			}
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

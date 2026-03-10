package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/gvm-tools/gvm/internal/auth"
	"github.com/gvm-tools/gvm/internal/config"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login <profile-name> <ssh|http>",
	Short: "Add or update authentication for a profile",
	Long:  "Set up SSH or HTTP/OAuth authentication for an existing profile.",
	Args:  cobra.ExactArgs(2),
	RunE:  runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		return fmt.Errorf("GVM not initialized — run 'gvm init' first")
	}

	name := args[0]
	method := args[1]

	if method != "ssh" && method != "http" {
		return fmt.Errorf("invalid auth method '%s' — must be 'ssh' or 'http'", method)
	}

	p, err := profile.Load(name)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)

	switch method {
	case "ssh":
		// Check for existing key
		exists, _ := auth.SSHKeyExists(name)
		if exists {
			keyPath, _ := auth.SSHKeyPath(name)
			if !promptConfirm(reader, fmt.Sprintf("SSH key already exists at %s. Regenerate? (y/N): ", keyPath)) {
				ui.Info("Keeping existing SSH key")
				return nil
			}
			// User confirmed — delete old key so setupSSH doesn't ask again
			auth.DeleteSSHKey(name)
		}

		if err := setupSSHForProfile(reader, p); err != nil {
			return err
		}

		if p.AuthMethod == profile.AuthHTTP {
			p.AuthMethod = profile.AuthBoth
		} else if p.AuthMethod != profile.AuthBoth {
			p.AuthMethod = profile.AuthSSH
		}

	case "http":
		if err := setupHTTPForProfile(p); err != nil {
			return err
		}

		if p.AuthMethod == profile.AuthSSH {
			p.AuthMethod = profile.AuthBoth
		} else if p.AuthMethod != profile.AuthBoth {
			p.AuthMethod = profile.AuthHTTP
		}
	}

	if err := p.Save(); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	fmt.Println()
	ui.Success("%s auth updated for profile '%s'", method, name)
	return nil
}

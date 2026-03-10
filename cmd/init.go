package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gvm-tools/gvm/internal/auth"
	"github.com/gvm-tools/gvm/internal/config"
	"github.com/gvm-tools/gvm/internal/git"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/gvm-tools/gvm/internal/shell"
	"github.com/gvm-tools/gvm/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive first-time setup wizard",
	Long:  "Initialize GVM: create config directory, set up shell hook, and create your first identity profile.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if config.Exists() {
		ui.Warn("GVM is already initialized at ~/.gvm/")
		fmt.Println("  Run 'gvm add <name>' to add a new profile.")
		fmt.Println("  Run 'gvm doctor' to check your setup.")
		return nil
	}

	fmt.Println(ui.Bold("Welcome to GVM!") + " Let's set up your first identity.\n")

	reader := bufio.NewReader(os.Stdin)

	// Step 1: Create directory structure
	if err := config.EnsureDirectories(); err != nil {
		return fmt.Errorf("creating config directories: %w", err)
	}

	// Step 2: Detect shell and save config
	detectedShell := shell.Detect()
	cfg := config.DefaultConfig(detectedShell.String())
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Step 3: Add .gvmrc to global gitignore
	if err := git.EnsureGlobalGitignore(); err != nil {
		ui.Warn("Could not update global gitignore: %v", err)
		fmt.Println("  Add '.gvmrc' to your global gitignore manually.")
	} else {
		ui.Success("Added .gvmrc to global gitignore")
	}

	// Step 4: Install shell hook
	fmt.Printf("\nDetected shell: %s\n", ui.Bold(detectedShell.String()))
	fmt.Printf("Install auto-switch hook to %s? (Y/n): ", detectedShell.ConfigFile())
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer == "" || answer == "y" || answer == "yes" {
		installed, err := shell.InstallHook(detectedShell)
		if err != nil {
			ui.Warn("Could not install shell hook: %v", err)
		} else if installed {
			ui.Success("Shell hook installed in %s", detectedShell.ConfigFile())
		} else {
			ui.Success("Shell hook already installed")
		}
	}

	// Step 5: Create first profile
	fmt.Println()
	p, err := promptForProfile(reader)
	if err != nil {
		return err
	}

	if err := p.Save(); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	// Activate the first profile
	if err := config.SetActive(p.Name); err != nil {
		return fmt.Errorf("setting active profile: %w", err)
	}

	ui.Success("Profile '%s' created and activated", p.Name)

	// Step 6: Offer to add another profile
	fmt.Println()
	if promptConfirm(reader, "Want to add another profile? (y/N): ") {
		fmt.Println()
		p2, err := promptForProfile(reader)
		if err != nil {
			ui.Warn("Could not create second profile: %v", err)
		} else {
			if err := p2.Save(); err != nil {
				ui.Warn("Could not save second profile: %v", err)
			} else {
				ui.Success("Profile '%s' created", p2.Name)
			}
		}
	}

	// Step 7: Run doctor
	fmt.Println()
	return runDoctor(cmd, nil)
}

func promptForProfile(reader *bufio.Reader) (*profile.Profile, error) {
	p := &profile.Profile{}

	p.Name = prompt(reader, "Profile name: ")
	if err := profile.ValidateName(p.Name); err != nil {
		return nil, err
	}

	exists, err := profile.Exists(p.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("profile '%s' already exists", p.Name)
	}

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

func setupSSHForProfile(reader *bufio.Reader, p *profile.Profile) error {
	// Check if key already exists
	exists, _ := auth.SSHKeyExists(p.Name)
	if exists {
		keyPath, _ := auth.SSHKeyPath(p.Name)
		fmt.Printf("SSH key already exists at %s. Overwrite? (y/N): ", keyPath)
		if !promptConfirm(reader, "") {
			p.SSHKeyPath, _ = auth.SSHKeyPath(p.Name)
			ui.Info("Keeping existing SSH key")

			// Still offer to copy/verify — user may need to add the key to the right account
			if promptConfirm(reader, "Copy public key and verify connection? (y/N): ") {
				if err := auth.CopyPublicKeyToClipboard(p.Name); err != nil {
					ui.Warn("Could not copy key: %v", err)
				}

				if p.GitHubUsername != "" {
					ui.Info("Make sure you're logged into GitHub as '%s' before adding the key", p.GitHubUsername)
				}
				if promptConfirmYes(reader, "Open GitHub to add your SSH key? (Y/n): ") {
					openGitHubSSHSettings()
				}

				fmt.Print("\nPress Enter when done...")
				reader.ReadString('\n')

				for {
					sp := ui.NewSpinner("Verifying SSH connection")
					if err := auth.VerifySSHConnection(p.SSHKeyPath, p.GitHubUsername); err != nil {
						sp.StopWithMessage(false, "SSH verification failed")
						ui.Warn("%v", err)
						fmt.Println()
						if promptConfirm(reader, "Retry verification? (y/N): ") {
							continue
						}
						ui.Info("You can verify later with 'gvm doctor'")
					} else {
						sp.StopWithMessage(true, fmt.Sprintf("Connected as %s", p.GitHubUsername))
					}
					break
				}
			}
			return nil
		}
	}

	sp := ui.NewSpinner("Generating SSH key for '" + p.Name + "'")
	keyPath, err := auth.GenerateSSHKey(p.Name)
	if err != nil {
		sp.Stop(false)
		return err
	}
	p.SSHKeyPath = keyPath
	sp.StopWithMessage(true, "SSH key generated")

	// Copy public key to clipboard
	if err := auth.CopyPublicKeyToClipboard(p.Name); err != nil {
		ui.Warn("Could not copy key: %v", err)
	}

	// Offer to open GitHub
	if p.GitHubUsername != "" {
		ui.Info("Make sure you're logged into GitHub as '%s' before adding the key", p.GitHubUsername)
	}
	if promptConfirmYes(reader, "Open GitHub to add your SSH key? (Y/n): ") {
		openGitHubSSHSettings()
	}

	fmt.Print("\nPress Enter when done...")
	reader.ReadString('\n')

	// Verify connection with retry loop
	for {
		sp = ui.NewSpinner("Verifying SSH connection")
		if err := auth.VerifySSHConnection(keyPath, p.GitHubUsername); err != nil {
			sp.StopWithMessage(false, "SSH verification failed")
			ui.Warn("%v", err)
			fmt.Println()
			if promptConfirm(reader, "Retry verification? (y/N): ") {
				continue
			}
			ui.Info("You can verify later with 'gvm doctor'")
		} else {
			sp.StopWithMessage(true, fmt.Sprintf("Connected as %s", p.GitHubUsername))
		}
		break
	}

	return nil
}

func setupHTTPForProfile(p *profile.Profile) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	clientID := cfg.GetGitHubClientID()
	token, err := auth.OAuthDeviceFlow(clientID)
	if err != nil {
		return err
	}

	if err := storeToken(p, token); err != nil {
		return err
	}

	return nil
}

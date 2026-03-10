package migrate

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DetectedIdentity represents an existing Git/GitHub identity found on the machine.
type DetectedIdentity struct {
	Source         string // "gitconfig", "ssh-config", "ssh-key", "gh-cli"
	Name           string // suggested profile name
	GitName        string
	GitEmail       string
	GitHubUsername string
	SSHKeyPath     string
	SSHHostAlias   string // from ~/.ssh/config Host entry
	GHToken        string // from gh CLI
	Description    string // human-readable description of what was found
}

// ScanAll runs all detection methods and returns found identities.
func ScanAll() []DetectedIdentity {
	var all []DetectedIdentity

	all = append(all, scanGitConfig()...)
	all = append(all, scanSSHConfig()...)
	all = append(all, scanSSHKeys()...)
	all = append(all, scanGHCLI()...)
	all = append(all, scanIncludeIf()...)

	return deduplicate(all)
}

// scanGitConfig reads the current global git config for user.name / user.email.
func scanGitConfig() []DetectedIdentity {
	name := gitConfigGlobal("user.name")
	email := gitConfigGlobal("user.email")

	if name == "" && email == "" {
		return nil
	}

	return []DetectedIdentity{{
		Source:      "gitconfig",
		Name:        suggestName(email, name),
		GitName:     name,
		GitEmail:    email,
		Description: fmt.Sprintf("Global git config: %s <%s>", name, email),
	}}
}

// scanSSHConfig parses ~/.ssh/config for GitHub-related Host entries.
func scanSSHConfig() []DetectedIdentity {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(home, ".ssh", "config"))
	if err != nil {
		return nil
	}

	var results []DetectedIdentity
	lines := strings.Split(string(data), "\n")

	var currentHost string
	var currentHostname string
	var currentIdentityFile string
	var currentUser string

	flush := func() {
		if currentHost == "" {
			return
		}
		// Only consider entries where HostName is explicitly github.com
		isGitHub := strings.EqualFold(currentHostname, "github.com")
		if !isGitHub && currentHostname == "" {
			// No explicit HostName — check if the Host alias is literally github.com
			isGitHub = strings.EqualFold(currentHost, "github.com")
		}

		if !isGitHub {
			currentHost = ""
			currentHostname = ""
			currentIdentityFile = ""
			currentUser = ""
			return
		}

		id := DetectedIdentity{
			Source:       "ssh-config",
			Name:        sanitizeName(currentHost),
			SSHHostAlias: currentHost,
			Description: fmt.Sprintf("SSH config: Host %s → %s", currentHost, currentHostname),
		}
		if currentIdentityFile != "" {
			id.SSHKeyPath = expandPath(currentIdentityFile)
			id.Description += fmt.Sprintf(" (key: %s)", currentIdentityFile)
		}
		if currentUser != "" {
			id.GitHubUsername = currentUser
		}

		results = append(results, id)

		currentHost = ""
		currentHostname = ""
		currentIdentityFile = ""
		currentUser = ""
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])

		switch key {
		case "host":
			flush()
			if val != "*" {
				currentHost = val
			}
		case "hostname":
			currentHostname = val
		case "identityfile":
			currentIdentityFile = val
		case "user":
			currentUser = val
		}
	}
	flush()

	return results
}

// scanSSHKeys looks for SSH keys in ~/.ssh/ that look GitHub-related.
func scanSSHKeys() []DetectedIdentity {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	sshDir := filepath.Join(home, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil
	}

	var results []DetectedIdentity
	seen := make(map[string]bool)

	for _, entry := range entries {
		name := entry.Name()
		// Skip public keys, known_hosts, config, and GVM-managed keys
		if strings.HasSuffix(name, ".pub") || name == "config" ||
			name == "known_hosts" || name == "authorized_keys" ||
			strings.HasPrefix(name, "gvm_") {
			continue
		}

		keyPath := filepath.Join(sshDir, name)
		info, err := entry.Info()
		if err != nil || info.IsDir() || info.Size() > 10000 {
			continue
		}

		// Check if it's actually a private key
		data, err := os.ReadFile(keyPath)
		if err != nil {
			continue
		}
		content := string(data)
		if !strings.Contains(content, "PRIVATE KEY") {
			continue
		}

		if seen[keyPath] {
			continue
		}
		seen[keyPath] = true

		// Read comment from the public key to find hints
		pubKeyComment := readPubKeyComment(keyPath + ".pub")

		id := DetectedIdentity{
			Source:      "ssh-key",
			SSHKeyPath:  keyPath,
			Name:        sanitizeName(strings.TrimPrefix(name, "id_")),
			Description: fmt.Sprintf("SSH key: %s", keyPath),
		}

		if pubKeyComment != "" {
			id.Description += fmt.Sprintf(" (comment: %s)", pubKeyComment)
			// If comment looks like an email, use it
			if strings.Contains(pubKeyComment, "@") {
				id.GitEmail = pubKeyComment
				id.Name = suggestName(pubKeyComment, "")
			}
		}

		results = append(results, id)
	}

	return results
}

// scanGHCLI checks for existing GitHub CLI authentication.
func scanGHCLI() []DetectedIdentity {
	// Check if gh is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return nil
	}

	// Get auth status
	out, err := exec.Command("gh", "auth", "status", "--hostname", "github.com").CombinedOutput()
	if err != nil {
		return nil
	}

	// Parse output for username and token
	output := string(out)
	var username string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Logged in to github.com") {
			// Extract "as <username>"
			if idx := strings.Index(line, " as "); idx >= 0 {
				rest := line[idx+4:]
				parts := strings.Fields(rest)
				if len(parts) > 0 {
					username = parts[0]
				}
			}
		}
	}

	if username == "" {
		return nil
	}

	// Try to get the token
	tokenOut, err := exec.Command("gh", "auth", "token", "--hostname", "github.com").Output()
	var token string
	if err == nil {
		token = strings.TrimSpace(string(tokenOut))
	}

	return []DetectedIdentity{{
		Source:         "gh-cli",
		Name:          sanitizeName(username),
		GitHubUsername: username,
		GHToken:        token,
		Description:    fmt.Sprintf("GitHub CLI: authenticated as %s", username),
	}}
}

// scanIncludeIf parses ~/.gitconfig for includeIf entries that point to
// directory-specific git configs with different identities.
func scanIncludeIf() []DetectedIdentity {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(home, ".gitconfig"))
	if err != nil {
		return nil
	}

	var results []DetectedIdentity
	lines := strings.Split(string(data), "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "[includeIf") {
			continue
		}

		// Extract the directory pattern
		dirPattern := ""
		if idx := strings.Index(line, "gitdir:"); idx >= 0 {
			dirPattern = strings.Trim(line[idx+7:], "\"]/ ")
		}

		// Find the path = ... line
		configPath := ""
		for j := i + 1; j < len(lines) && j < i+5; j++ {
			l := strings.TrimSpace(lines[j])
			if strings.HasPrefix(l, "path") {
				parts := strings.SplitN(l, "=", 2)
				if len(parts) == 2 {
					configPath = strings.TrimSpace(parts[1])
				}
				break
			}
			if strings.HasPrefix(l, "[") {
				break
			}
		}

		if configPath == "" {
			continue
		}

		// Read the included config for name/email
		includedPath := expandPath(configPath)
		incData, err := os.ReadFile(includedPath)
		if err != nil {
			continue
		}

		incName, incEmail := parseGitConfigNameEmail(string(incData))
		if incName == "" && incEmail == "" {
			continue
		}

		results = append(results, DetectedIdentity{
			Source:      "includeIf",
			Name:        suggestName(incEmail, incName),
			GitName:     incName,
			GitEmail:    incEmail,
			Description: fmt.Sprintf("includeIf: %s → %s <%s> (dir: %s)", configPath, incName, incEmail, dirPattern),
		})
	}

	return results
}

func gitConfigGlobal(key string) string {
	out, err := exec.Command("git", "config", "--global", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func readPubKeyComment(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	parts := strings.Fields(string(data))
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return ""
}

func parseGitConfigNameEmail(content string) (string, string) {
	var name, email string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name = strings.TrimSpace(parts[1])
			}
		}
		if strings.HasPrefix(line, "email") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				email = strings.TrimSpace(parts[1])
			}
		}
	}
	return name, email
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

func suggestName(email, name string) string {
	// Try to extract a meaningful name from email
	if email != "" {
		parts := strings.Split(email, "@")
		if len(parts) == 2 {
			domain := parts[1]
			// Use domain hint for well-known providers
			switch {
			case strings.Contains(domain, "gmail") || strings.Contains(domain, "yahoo") ||
				strings.Contains(domain, "hotmail") || strings.Contains(domain, "outlook"):
				return sanitizeName("personal")
			default:
				// Use the domain name without TLD
				domParts := strings.Split(domain, ".")
				if len(domParts) > 0 {
					return sanitizeName(domParts[0])
				}
			}
		}
	}
	if name != "" {
		return sanitizeName(strings.Split(name, " ")[0])
	}
	return "imported"
}

func sanitizeName(s string) string {
	s = strings.ToLower(s)
	var result []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result = append(result, c)
		} else if c == ' ' || c == '_' || c == '.' {
			result = append(result, '-')
		}
	}
	name := string(result)
	if len(name) > 32 {
		name = name[:32]
	}
	if name == "" {
		return "imported"
	}
	// Must start with letter or number
	if name[0] == '-' {
		name = "x" + name
	}
	return name
}

func deduplicate(identities []DetectedIdentity) []DetectedIdentity {
	seen := make(map[string]bool)
	var result []DetectedIdentity

	for _, id := range identities {
		// Deduplicate by email + ssh key path combo
		key := id.GitEmail + "|" + id.SSHKeyPath + "|" + id.GitHubUsername
		if key == "||" {
			// No useful dedup key, keep it
			result = append(result, id)
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, id)
	}

	return result
}

// ToJSON returns a JSON representation (useful for debugging).
func (d DetectedIdentity) ToJSON() string {
	b, _ := json.MarshalIndent(d, "", "  ")
	return string(b)
}

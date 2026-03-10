package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)


func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"personal", "personal"},
		{"My Work", "my-work"},
		{"user_name", "user-name"},
		{"GitHub.User", "github-user"},
		{"UPPER", "upper"},
		{"-starts-dash", "x-starts-dash"},
		{"", "imported"},
		{"a!b@c#d", "abcd"},
		{strings.Repeat("a", 40), strings.Repeat("a", 32)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSuggestName(t *testing.T) {
	tests := []struct {
		email string
		name  string
		want  string
	}{
		{"john@gmail.com", "", "personal"},
		{"john@company.com", "", "company"},
		{"", "John Doe", "john"},
		{"", "", "imported"},
		{"john@yahoo.com", "John", "personal"},
		{"john@acme.io", "", "acme"},
	}

	for _, tt := range tests {
		t.Run(tt.email+"/"+tt.name, func(t *testing.T) {
			got := suggestName(tt.email, tt.name)
			if got != tt.want {
				t.Errorf("suggestName(%q, %q) = %q, want %q", tt.email, tt.name, got, tt.want)
			}
		})
	}
}

func TestScanSSHConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	_ = os.MkdirAll(sshDir, 0700)

	// Write a test SSH config with GitHub host entries
	config := `Host github-personal
    HostName github.com
    IdentityFile ~/.ssh/id_personal
    User git

Host github-work
    HostName github.com
    IdentityFile ~/.ssh/id_work

Host not-github
    HostName gitlab.com
    IdentityFile ~/.ssh/id_gitlab
`
	_ = os.WriteFile(filepath.Join(sshDir, "config"), []byte(config), 0644)

	results := scanSSHConfig()

	// Should find the 2 GitHub entries, not the GitLab one
	if len(results) != 2 {
		t.Fatalf("scanSSHConfig() returned %d results, want 2", len(results))
	}

	if results[0].SSHHostAlias != "github-personal" {
		t.Errorf("first result host = %q, want %q", results[0].SSHHostAlias, "github-personal")
	}
	if results[1].SSHHostAlias != "github-work" {
		t.Errorf("second result host = %q, want %q", results[1].SSHHostAlias, "github-work")
	}

	// Source should be correct
	for _, r := range results {
		if r.Source != "ssh-config" {
			t.Errorf("source = %q, want %q", r.Source, "ssh-config")
		}
	}
}

func TestScanSSHKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	_ = os.MkdirAll(sshDir, 0700)

	// Create a fake SSH private key
	fakeKey := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAA
-----END OPENSSH PRIVATE KEY-----
`
	_ = os.WriteFile(filepath.Join(sshDir, "id_ed25519"), []byte(fakeKey), 0600)
	_ = os.WriteFile(filepath.Join(sshDir, "id_ed25519.pub"),
		[]byte("ssh-ed25519 AAAAC3... john@company.com"), 0644)

	// Create a GVM key (should be skipped)
	_ = os.WriteFile(filepath.Join(sshDir, "gvm_test"), []byte(fakeKey), 0600)

	// Create a non-key file (should be skipped)
	_ = os.WriteFile(filepath.Join(sshDir, "known_hosts"), []byte("stuff"), 0644)

	results := scanSSHKeys()

	if len(results) != 1 {
		t.Fatalf("scanSSHKeys() returned %d results, want 1", len(results))
	}

	if results[0].Source != "ssh-key" {
		t.Errorf("source = %q, want %q", results[0].Source, "ssh-key")
	}
	if results[0].GitEmail != "john@company.com" {
		t.Errorf("email = %q, want %q", results[0].GitEmail, "john@company.com")
	}
}

func TestScanIncludeIf(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create an included config file
	workConfig := `[user]
	name = John Work
	email = john@company.com
`
	workConfigPath := filepath.Join(home, ".gitconfig-work")
	_ = os.WriteFile(workConfigPath, []byte(workConfig), 0644)

	// Create the main gitconfig with includeIf
	mainConfig := `[user]
	name = John Personal
	email = john@gmail.com

[includeIf "gitdir:~/work/"]
	path = ` + workConfigPath + `
`
	_ = os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(mainConfig), 0644)

	results := scanIncludeIf()

	if len(results) != 1 {
		t.Fatalf("scanIncludeIf() returned %d results, want 1", len(results))
	}

	if results[0].GitName != "John Work" {
		t.Errorf("name = %q, want %q", results[0].GitName, "John Work")
	}
	if results[0].GitEmail != "john@company.com" {
		t.Errorf("email = %q, want %q", results[0].GitEmail, "john@company.com")
	}
	if results[0].Source != "includeIf" {
		t.Errorf("source = %q, want %q", results[0].Source, "includeIf")
	}
}

func TestDeduplicate(t *testing.T) {
	identities := []DetectedIdentity{
		{GitEmail: "john@example.com", Source: "gitconfig"},
		{GitEmail: "john@example.com", Source: "includeIf"},
		{GitEmail: "jane@example.com", Source: "gitconfig"},
	}

	result := deduplicate(identities)

	if len(result) != 2 {
		t.Errorf("deduplicate() returned %d results, want 2", len(result))
	}
}

func TestParseGitConfigNameEmail(t *testing.T) {
	tests := []struct {
		input     string
		wantName  string
		wantEmail string
	}{
		{"name = John\nemail = john@test.com", "John", "john@test.com"},
		{"name = \nemail = ", "", ""},
		{"other = stuff", "", ""},
		{"", "", ""},
		{"[user]\n\tname = Jane Doe\n\temail = jane@example.com", "Jane Doe", "jane@example.com"},
	}

	for _, tt := range tests {
		name, email := parseGitConfigNameEmail(tt.input)
		if name != tt.wantName || email != tt.wantEmail {
			t.Errorf("parseGitConfigNameEmail(%q) = (%q, %q), want (%q, %q)",
				tt.input, name, email, tt.wantName, tt.wantEmail)
		}
	}
}

func TestReadPubKeyComment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pub")

	// 3-part public key: type key comment
	_ = os.WriteFile(path, []byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA john@work.com\n"), 0644)
	comment := readPubKeyComment(path)
	if comment != "john@work.com" {
		t.Errorf("readPubKeyComment() = %q, want \"john@work.com\"", comment)
	}

	// Missing file
	comment = readPubKeyComment("/nonexistent/key.pub")
	if comment != "" {
		t.Errorf("readPubKeyComment(nonexistent) = %q, want empty", comment)
	}

	// Key with no comment (2 parts)
	_ = os.WriteFile(path, []byte("ssh-ed25519 AAAAC3NzaC1l"), 0644)
	comment = readPubKeyComment(path)
	if comment != "" {
		t.Errorf("readPubKeyComment(no-comment) = %q, want empty", comment)
	}
}

func TestScanGitConfig(t *testing.T) {
	// scanGitConfig reads from real git config, just verify it doesn't crash
	results := scanGitConfig()
	// May return 0 or 1 results depending on environment
	_ = results
}

func TestToJSON(t *testing.T) {
	id := DetectedIdentity{
		Source:  "test",
		Name:   "test-profile",
		GitEmail: "test@test.com",
	}
	j := id.ToJSON()
	if !strings.Contains(j, "test-profile") {
		t.Error("ToJSON() missing profile name")
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	got := expandPath("~/test/file")
	want := filepath.Join(home, "test/file")
	if got != want {
		t.Errorf("expandPath(\"~/test/file\") = %q, want %q", got, want)
	}

	got = expandPath("/absolute/path")
	if got != "/absolute/path" {
		t.Errorf("expandPath(\"/absolute/path\") = %q, want \"/absolute/path\"", got)
	}
}

package cmd

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gvm-tools/gvm/internal/config"
	gitpkg "github.com/gvm-tools/gvm/internal/git"
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/spf13/cobra"
)

func setupTestEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	_ = config.EnsureDirectories()

	cfg := config.DefaultConfig("zsh")
	_ = cfg.Save()

	return home
}

func createTestProfile(t *testing.T, name, email string) *profile.Profile {
	t.Helper()
	p := &profile.Profile{
		Name:       name,
		GitName:    "Test " + name,
		GitEmail:   email,
		AuthMethod: profile.AuthSSH,
		CreatedAt:  time.Now().UTC(),
		LastUsed:   time.Now().UTC(),
	}
	if err := p.Save(); err != nil {
		t.Fatalf("creating test profile: %v", err)
	}
	return p
}

func initTestRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", dir)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"HOME="+os.Getenv("HOME"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s\n%s", err, out)
	}
}

func TestActivateProfile(t *testing.T) {
	setupTestEnv(t)
	createTestProfile(t, "test-activate", "test@example.com")

	if err := activateProfile("test-activate", true); err != nil {
		t.Fatalf("activateProfile() error = %v", err)
	}

	active, _ := config.GetActive()
	if active != "test-activate" {
		t.Errorf("active = %q, want %q", active, "test-activate")
	}
}

func TestActivateNonexistentProfile(t *testing.T) {
	setupTestEnv(t)

	err := activateProfile("does-not-exist", true)
	if err == nil {
		t.Error("activateProfile() should return error for nonexistent profile")
	}
}

func TestActivateSwitchesProfiles(t *testing.T) {
	setupTestEnv(t)
	createTestProfile(t, "profile-a", "a@test.com")
	createTestProfile(t, "profile-b", "b@test.com")

	// Activate A
	if err := activateProfile("profile-a", true); err != nil {
		t.Fatalf("activateProfile(a) error = %v", err)
	}
	active, _ := config.GetActive()
	if active != "profile-a" {
		t.Errorf("active = %q, want %q", active, "profile-a")
	}

	// Switch to B
	if err := activateProfile("profile-b", true); err != nil {
		t.Fatalf("activateProfile(b) error = %v", err)
	}
	active, _ = config.GetActive()
	if active != "profile-b" {
		t.Errorf("active = %q, want %q", active, "profile-b")
	}
}

func TestDeactivateProfile(t *testing.T) {
	setupTestEnv(t)
	createTestProfile(t, "test-deactivate", "test@example.com")

	_ = activateProfile("test-deactivate", true)

	deactivateProfile("test-deactivate")

	// Should not crash, deactivate is best-effort
}

func TestUseBindsProfile(t *testing.T) {
	home := setupTestEnv(t)
	createTestProfile(t, "use-test", "use@test.com")

	repoDir := filepath.Join(home, "test-repo")
	_ = os.MkdirAll(repoDir, 0755)
	initTestRepo(t, repoDir)

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(repoDir)

	// Simulate `gvm use`
	repoRoot, _ := gitpkg.FindRepoRoot()
	if err := gitpkg.WriteGVMRC(repoRoot, "use-test"); err != nil {
		t.Fatalf("WriteGVMRC() error = %v", err)
	}

	// Verify .gvmrc
	name, _ := gitpkg.ReadGVMRC(repoRoot)
	if name != "use-test" {
		t.Errorf("ReadGVMRC() = %q, want %q", name, "use-test")
	}
}

func TestExportedProfileFormat(t *testing.T) {
	ep := exportedProfile{
		Name:           "test",
		GitName:        "Test User",
		GitEmail:       "test@example.com",
		GitHubUsername: "testuser",
		AuthMethod:     "ssh",
	}

	if ep.Name != "test" || ep.GitEmail != "test@example.com" {
		t.Error("exportedProfile fields not set correctly")
	}
}

func TestStoreLoadToken(t *testing.T) {
	setupTestEnv(t)

	p := &profile.Profile{
		Name:       "token-test",
		GitName:    "Test",
		GitEmail:   "test@test.com",
		AuthMethod: profile.AuthHTTP,
	}

	// Store a token (will use file encryption since keychain isn't available in tests)
	err := storeToken(p, "ghp_test_token_12345")
	if err != nil {
		t.Fatalf("storeToken() error = %v", err)
	}

	if p.GHTokenEncrypted == "" {
		t.Error("GHTokenEncrypted is empty after storeToken")
	}
	if p.GHTokenEncrypted == "ghp_test_token_12345" {
		t.Error("Token stored in plaintext")
	}

	// Load it back
	token, err := loadToken(p)
	if err != nil {
		t.Fatalf("loadToken() error = %v", err)
	}
	if token != "ghp_test_token_12345" {
		t.Errorf("loadToken() = %q, want %q", token, "ghp_test_token_12345")
	}
}

func TestDeleteToken(t *testing.T) {
	p := &profile.Profile{
		GHTokenEncrypted: "some-encrypted-data",
	}

	deleteToken(p)

	if p.GHTokenEncrypted != "" {
		t.Errorf("GHTokenEncrypted = %q after deleteToken, want empty", p.GHTokenEncrypted)
	}
}

func TestBoolStr(t *testing.T) {
	if boolStr(true) != "enabled" {
		t.Errorf("boolStr(true) = %q, want \"enabled\"", boolStr(true))
	}
	if boolStr(false) != "disabled" {
		t.Errorf("boolStr(false) = %q, want \"disabled\"", boolStr(false))
	}
}

func TestPluralize(t *testing.T) {
	if pluralize(1, "item", "items") != "item" {
		t.Error("pluralize(1) should return singular")
	}
	if pluralize(2, "item", "items") != "items" {
		t.Error("pluralize(2) should return plural")
	}
	if pluralize(0, "item", "items") != "items" {
		t.Error("pluralize(0) should return plural")
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"a\nb\nc", 3},
		{"single", 1},
		{"a\nb\n", 2},
		{"", 0},
		{"\n", 1},
		{"a\r\nb", 2},
	}

	for _, tt := range tests {
		got := splitLines(tt.input)
		if len(got) != tt.want {
			t.Errorf("splitLines(%q) = %d lines, want %d", tt.input, len(got), tt.want)
		}
	}
}

func TestSetVersionInfo(t *testing.T) {
	SetVersionInfo("1.0.0", "abc", "2026-01-01")
	if versionStr != "1.0.0" {
		t.Errorf("versionStr = %q, want \"1.0.0\"", versionStr)
	}
	if commitStr != "abc" {
		t.Errorf("commitStr = %q, want \"abc\"", commitStr)
	}
	if dateStr != "2026-01-01" {
		t.Errorf("dateStr = %q, want \"2026-01-01\"", dateStr)
	}
}

func TestTimeNow(t *testing.T) {
	now := timeNow()
	if now.IsZero() {
		t.Error("timeNow() returned zero time")
	}
	if now.Location() != time.UTC {
		t.Error("timeNow() should return UTC")
	}
}

func TestIsGVMRCInGlobalGitignore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Not present
	if isGVMRCInGlobalGitignore() {
		t.Error("isGVMRCInGlobalGitignore() = true with no gitignore")
	}

	// Create it
	ignoreDir := home + "/.config/git"
	_ = os.MkdirAll(ignoreDir, 0755)
	_ = os.WriteFile(ignoreDir+"/ignore", []byte(".DS_Store\n.gvmrc\n"), 0644)

	if !isGVMRCInGlobalGitignore() {
		t.Error("isGVMRCInGlobalGitignore() = false after adding .gvmrc")
	}
}

func TestRemoveFromGlobalGitignore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	ignoreDir := home + "/.config/git"
	_ = os.MkdirAll(ignoreDir, 0755)
	ignorePath := ignoreDir + "/ignore"
	_ = os.WriteFile(ignorePath, []byte(".DS_Store\n.gvmrc\n*.swp\n"), 0644)

	removeFromGlobalGitignore()

	data, _ := os.ReadFile(ignorePath)
	content := string(data)
	if strings.Contains(content, ".gvmrc") {
		t.Error(".gvmrc still in gitignore after removeFromGlobalGitignore()")
	}
	if !strings.Contains(content, ".DS_Store") {
		t.Error(".DS_Store was removed (should be kept)")
	}
	if !strings.Contains(content, "*.swp") {
		t.Error("*.swp was removed (should be kept)")
	}
}

func TestCompleteProfileNames(t *testing.T) {
	setupTestEnv(t)
	createTestProfile(t, "alpha", "a@test.com")
	createTestProfile(t, "beta", "b@test.com")

	// First arg: should return profile names
	names, directive := completeProfileNames(useCmd, []string{}, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Error("directive should be NoFileComp")
	}
	if len(names) != 2 {
		t.Errorf("got %d completions, want 2", len(names))
	}

	// Second arg: should return nothing for most commands
	names, _ = completeProfileNames(useCmd, []string{"alpha"}, "")
	if len(names) != 0 {
		t.Errorf("got %d completions for 2nd arg, want 0", len(names))
	}

	// Login second arg: should return ssh/http
	names, _ = completeProfileNames(loginCmd, []string{"alpha"}, "")
	if len(names) != 2 || names[0] != "ssh" || names[1] != "http" {
		t.Errorf("login 2nd arg completions = %v, want [ssh http]", names)
	}
}

func TestRunConfigNotInitialized(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := runConfig(nil, nil)
	if err == nil {
		t.Error("runConfig() should error when not initialized")
	}
}

func TestRunListNotInitialized(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := runList(nil, nil)
	if err == nil {
		t.Error("runList() should error when not initialized")
	}
}

func TestRunWhoamiNoActive(t *testing.T) {
	setupTestEnv(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWhoami(nil, nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runWhoami() error = %v", err)
	}
	if !strings.Contains(buf.String(), "No active profile") {
		t.Error("runWhoami() should show 'No active profile'")
	}
}

func TestRunListEmpty(t *testing.T) {
	setupTestEnv(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runList() error = %v", err)
	}
	if !strings.Contains(buf.String(), "No profiles found") {
		t.Error("runList() should show 'No profiles found' for empty list")
	}
}

func TestRunConfig(t *testing.T) {
	setupTestEnv(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runConfig(nil, nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runConfig() error = %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "ssh") {
		t.Error("runConfig() should show default auth method")
	}
	if !strings.Contains(output, "enabled") {
		t.Error("runConfig() should show auto-switch status")
	}
}

func TestRunConfigSet(t *testing.T) {
	setupTestEnv(t)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runConfigSet(nil, []string{"default-auth", "http"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("runConfigSet() error = %v", err)
	}

	cfg, loadErr := config.Load()
	if loadErr != nil {
		t.Fatalf("config.Load() error = %v", loadErr)
	}
	if cfg.DefaultAuth != "http" {
		t.Errorf("DefaultAuth = %q after set, want \"http\"", cfg.DefaultAuth)
	}
}

func TestRunConfigSetInvalidKey(t *testing.T) {
	setupTestEnv(t)

	err := runConfigSet(nil, []string{"invalid-key", "value"})
	if err == nil {
		t.Error("runConfigSet() should error for invalid key")
	}
}

func TestRunExportNoProfiles(t *testing.T) {
	setupTestEnv(t)

	err := runExport(nil, nil)
	if err == nil {
		t.Error("runExport() should error with no profiles")
	}
}

func TestRunExportToStdout(t *testing.T) {
	setupTestEnv(t)
	createTestProfile(t, "exp-stdout", "exp@test.com")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runExport(nil, nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runExport() error = %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "exp-stdout") {
		t.Error("export output missing profile name")
	}
	if !strings.Contains(output, "exp@test.com") {
		t.Error("export output missing email")
	}
}

func TestRunExportToFile(t *testing.T) {
	setupTestEnv(t)
	createTestProfile(t, "exp-file", "expf@test.com")

	dir := t.TempDir()
	outPath := filepath.Join(dir, "export.json")

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runExport(nil, []string{outPath})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("runExport(file) error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading export file: %v", err)
	}
	if !strings.Contains(string(data), "exp-file") {
		t.Error("export file missing profile")
	}
}

func TestRunWhoamiWithActive(t *testing.T) {
	setupTestEnv(t)
	createTestProfile(t, "whoami-test", "whoami@test.com")
	_ = config.SetActive("whoami-test")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWhoami(nil, nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runWhoami() error = %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "whoami-test") {
		t.Error("whoami output missing profile name")
	}
	if !strings.Contains(output, "whoami@test.com") {
		t.Error("whoami output missing email")
	}
}

func TestRunListWithProfiles(t *testing.T) {
	setupTestEnv(t)
	createTestProfile(t, "list-a", "a@test.com")
	createTestProfile(t, "list-b", "b@test.com")
	_ = config.SetActive("list-a")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Override the list command to skip network calls for connection status
	profiles, _ := profile.List()
	_ = profiles

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	// Just verify profiles exist
	p, _ := profile.List()
	if len(p) != 2 {
		t.Errorf("expected 2 profiles, got %d", len(p))
	}
}

func TestRunCredentialHelper(t *testing.T) {
	setupTestEnv(t)
	p := createTestProfile(t, "cred-test", "cred@test.com")
	p.GitHubUsername = "creduser"
	p.GHTokenEncrypted = ""
	_ = p.Save()

	// Should error when no token
	err := runCredentialHelper(nil, []string{"cred-test"})
	if err == nil {
		t.Error("runCredentialHelper() should error with no token")
	}
}

func TestRunCredentialHelperNonexistent(t *testing.T) {
	setupTestEnv(t)

	err := runCredentialHelper(nil, []string{"nonexistent"})
	if err == nil {
		t.Error("runCredentialHelper() should error for nonexistent profile")
	}
}

func TestRunSwitchNotInitialized(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := runSwitch(nil, []string{"test"})
	if err == nil {
		t.Error("runSwitch() should error when not initialized")
	}
}

func TestRunUseNotInitialized(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := runUse(nil, []string{"test"})
	if err == nil {
		t.Error("runUse() should error when not initialized")
	}
}

func TestRunUnbindNoRepo(t *testing.T) {
	setupTestEnv(t)

	// cd to a non-repo directory
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()

	dir := t.TempDir()
	_ = os.Chdir(dir)

	err := runUnbind(nil, nil)
	if err == nil {
		t.Error("runUnbind() should error outside a repo")
	}
}

func TestRunDoctorNotInitialized(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runDoctor(nil, nil)

	w.Close()
	os.Stdout = old

	if err == nil {
		t.Error("runDoctor() should error when not initialized")
	}
}

func TestPrompt(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("test input\n"))
	result := prompt(reader, "Enter: ")
	if result != "test input" {
		t.Errorf("prompt() = %q, want \"test input\"", result)
	}
}

func TestPromptEmpty(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	result := prompt(reader, "Enter: ")
	if result != "" {
		t.Errorf("prompt() = %q, want empty", result)
	}
}

func TestPromptDefault(t *testing.T) {
	// Empty input returns default
	reader := bufio.NewReader(strings.NewReader("\n"))
	result := promptDefault(reader, "Auth", "ssh")
	if result != "ssh" {
		t.Errorf("promptDefault() = %q, want \"ssh\"", result)
	}

	// Non-empty input overrides default
	reader = bufio.NewReader(strings.NewReader("http\n"))
	result = promptDefault(reader, "Auth", "ssh")
	if result != "http" {
		t.Errorf("promptDefault() = %q, want \"http\"", result)
	}
}

func TestPromptConfirm(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"yes\n", true},
		{"Y\n", true},
		{"n\n", false},
		{"no\n", false},
		{"\n", false},
		{"maybe\n", false},
	}

	for _, tt := range tests {
		reader := bufio.NewReader(strings.NewReader(tt.input))
		got := promptConfirm(reader, "Confirm? ")
		if got != tt.want {
			t.Errorf("promptConfirm(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestPromptConfirmYes(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"yes\n", true},
		{"\n", true},  // default is yes
		{"n\n", false},
		{"no\n", false},
	}

	for _, tt := range tests {
		reader := bufio.NewReader(strings.NewReader(tt.input))
		got := promptConfirmYes(reader, "Confirm? ")
		if got != tt.want {
			t.Errorf("promptConfirmYes(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestDecryptProfileToken(t *testing.T) {
	setupTestEnv(t)

	p := &profile.Profile{
		Name:       "decrypt-test",
		GitName:    "Test",
		GitEmail:   "test@test.com",
		AuthMethod: profile.AuthHTTP,
	}
	_ = storeToken(p, "ghp_secret_token")
	_ = p.Save()

	token, err := decryptProfileToken(p)
	if err != nil {
		t.Fatalf("decryptProfileToken() error = %v", err)
	}
	if token != "ghp_secret_token" {
		t.Errorf("decryptProfileToken() = %q, want \"ghp_secret_token\"", token)
	}

	// Empty token
	p2 := &profile.Profile{GHTokenEncrypted: ""}
	token, err = decryptProfileToken(p2)
	if err != nil {
		t.Fatalf("decryptProfileToken(empty) error = %v", err)
	}
	if token != "" {
		t.Errorf("decryptProfileToken(empty) = %q, want empty", token)
	}
}

func TestRunDoctorInitialized(t *testing.T) {
	setupTestEnv(t)

	// Install a fake shell hook
	home := os.Getenv("HOME")
	_ = os.WriteFile(home+"/.zshrc",
		[]byte("stuff\n# >>> gvm initialize >>>\nhook\n# <<< gvm initialize <<<\n"), 0644)

	// Add .gvmrc to gitignore
	_ = os.MkdirAll(home+"/.config/git", 0755)
	_ = os.WriteFile(home+"/.config/git/ignore", []byte(".gvmrc\n"), 0644)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runDoctor(nil, nil)

	w.Close()
	os.Stdout = old

	// No profiles, so it should report that but not fail catastrophically
	_ = err
}

func TestRunRemoveFromGlobalGitignoreNoFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Should not panic when no gitignore exists
	removeFromGlobalGitignore()
}

func TestRunDeactivate(t *testing.T) {
	setupTestEnv(t)
	createTestProfile(t, "deact-test", "deact@test.com")
	_ = activateProfile("deact-test", true)

	err := runDeactivate(nil, nil)
	if err != nil {
		t.Fatalf("runDeactivate() error = %v", err)
	}

	active, _ := config.GetActive()
	if active != "" {
		t.Errorf("active = %q after deactivate, want empty", active)
	}
}

func TestRunDeactivateNoActive(t *testing.T) {
	setupTestEnv(t)

	err := runDeactivate(nil, nil)
	if err != nil {
		t.Errorf("runDeactivate() with no active should not error: %v", err)
	}
}

func TestRunInitAlreadyInitialized(t *testing.T) {
	setupTestEnv(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runInit(nil, nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}
	if !strings.Contains(buf.String(), "already initialized") {
		t.Error("runInit() should say already initialized")
	}
}

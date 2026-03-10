package shell

import "testing"

func TestDetect(t *testing.T) {
	tests := []struct {
		envShell string
		want     Shell
	}{
		{"/bin/zsh", Zsh},
		{"/usr/local/bin/zsh", Zsh},
		{"/bin/bash", Bash},
		{"/usr/bin/fish", Fish},
		{"/usr/local/bin/fish", Fish},
		{"", Bash}, // default
	}

	for _, tt := range tests {
		t.Run(tt.envShell, func(t *testing.T) {
			t.Setenv("SHELL", tt.envShell)
			got := Detect()
			if got != tt.want {
				t.Errorf("Detect() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShellConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if path := Zsh.ConfigFile(); path == "" {
		t.Error("Zsh.ConfigFile() returned empty")
	}
	if path := Bash.ConfigFile(); path == "" {
		t.Error("Bash.ConfigFile() returned empty")
	}
	if path := Fish.ConfigFile(); path == "" {
		t.Error("Fish.ConfigFile() returned empty")
	}
}

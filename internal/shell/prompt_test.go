package shell

import (
	"strings"
	"testing"
)

func TestStarshipConfig(t *testing.T) {
	config := StarshipConfig()
	if !strings.Contains(config, "[custom.gvm]") {
		t.Error("StarshipConfig missing [custom.gvm] section")
	}
	if !strings.Contains(config, "bold purple") {
		t.Error("StarshipConfig missing style")
	}
	if !strings.Contains(config, "~/.gvm/active") {
		t.Error("StarshipConfig missing active file reference")
	}
}

func TestP10kSnippet(t *testing.T) {
	snippet := P10kSnippet()
	if !strings.Contains(snippet, "POWERLEVEL9K") {
		t.Error("P10kSnippet missing POWERLEVEL9K reference")
	}
	if !strings.Contains(snippet, "prompt_gvm_profile") {
		t.Error("P10kSnippet missing function definition")
	}
}

func TestOhMyZshSnippet(t *testing.T) {
	snippet := OhMyZshSnippet()
	if !strings.Contains(snippet, "RPROMPT") {
		t.Error("OhMyZshSnippet missing RPROMPT reference")
	}
	if !strings.Contains(snippet, "gvm_prompt_info") {
		t.Error("OhMyZshSnippet missing gvm_prompt_info reference")
	}
}

func TestShellString(t *testing.T) {
	tests := []struct {
		shell Shell
		want  string
	}{
		{Zsh, "zsh"},
		{Bash, "bash"},
		{Fish, "fish"},
	}

	for _, tt := range tests {
		if got := tt.shell.String(); got != tt.want {
			t.Errorf("%v.String() = %q, want %q", tt.shell, got, tt.want)
		}
	}
}

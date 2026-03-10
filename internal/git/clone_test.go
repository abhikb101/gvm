package git

import "testing"

func TestRepoNameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"git@github.com:user/repo.git", "repo"},
		{"git@github.com:user/repo", "repo"},
		{"https://github.com/user/repo.git", "repo"},
		{"https://github.com/user/repo", "repo"},
		{"ssh://git@github.com/user/repo.git", "repo"},
		{"git@github.com:org/my-project.git", "my-project"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := repoNameFromURL(tt.url)
			if got != tt.want {
				t.Errorf("repoNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestIsSSHURL(t *testing.T) {
	if !isSSHURL("git@github.com:user/repo.git") {
		t.Error("isSSHURL should be true for git@ URLs")
	}
	if !isSSHURL("ssh://git@github.com/repo") {
		t.Error("isSSHURL should be true for ssh:// URLs")
	}
	if isSSHURL("https://github.com/user/repo") {
		t.Error("isSSHURL should be false for https URLs")
	}
}

func TestIsHTTPSURL(t *testing.T) {
	if !isHTTPSURL("https://github.com/user/repo") {
		t.Error("isHTTPSURL should be true for https URLs")
	}
	if !isHTTPSURL("http://github.com/user/repo") {
		t.Error("isHTTPSURL should be true for http URLs")
	}
	if isHTTPSURL("git@github.com:user/repo") {
		t.Error("isHTTPSURL should be false for SSH URLs")
	}
}

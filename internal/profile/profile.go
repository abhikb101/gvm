package profile

import "time"

// AuthMethod represents how a profile authenticates with GitHub.
type AuthMethod string

const (
	AuthSSH  AuthMethod = "ssh"
	AuthHTTP AuthMethod = "http"
	AuthBoth AuthMethod = "both"
)

// Profile represents a named Git/GitHub identity.
type Profile struct {
	Name             string     `json:"name"`
	GitName          string     `json:"git_name"`
	GitEmail         string     `json:"git_email"`
	GitHubUsername   string     `json:"github_username"`
	AuthMethod       AuthMethod `json:"auth_method"`
	SSHKeyPath       string     `json:"ssh_key_path,omitempty"`
	GHTokenEncrypted string     `json:"gh_token_encrypted,omitempty"`
	SigningKey        string     `json:"signing_key,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	LastUsed         time.Time  `json:"last_used"`
}

// HasSSH returns true if this profile is configured for SSH auth.
func (p *Profile) HasSSH() bool {
	return p.AuthMethod == AuthSSH || p.AuthMethod == AuthBoth
}

// HasHTTP returns true if this profile is configured for HTTP/OAuth auth.
func (p *Profile) HasHTTP() bool {
	return p.AuthMethod == AuthHTTP || p.AuthMethod == AuthBoth
}

// AuthDisplay returns a human-readable auth method string.
func (p *Profile) AuthDisplay() string {
	switch p.AuthMethod {
	case AuthBoth:
		return "ssh+http"
	default:
		return string(p.AuthMethod)
	}
}

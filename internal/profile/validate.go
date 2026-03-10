package profile

import (
	"fmt"
	"regexp"
	"strings"
)

const maxNameLength = 32

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// Validate checks that all required fields are present and well-formed.
func (p *Profile) Validate() error {
	if err := ValidateName(p.Name); err != nil {
		return err
	}

	if strings.TrimSpace(p.GitName) == "" {
		return fmt.Errorf("git name cannot be empty")
	}

	if !emailPattern.MatchString(p.GitEmail) {
		return fmt.Errorf("invalid email address '%s'", p.GitEmail)
	}

	if p.AuthMethod != AuthSSH && p.AuthMethod != AuthHTTP && p.AuthMethod != AuthBoth {
		return fmt.Errorf("invalid auth method '%s' — must be ssh, http, or both", p.AuthMethod)
	}

	return nil
}

// ValidateName checks that a profile name is valid.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	if len(name) > maxNameLength {
		return fmt.Errorf("profile name too long (max %d characters)", maxNameLength)
	}

	if !namePattern.MatchString(name) {
		return fmt.Errorf("profile name '%s' is invalid — use lowercase letters, numbers, and hyphens only (must start with letter or number)", name)
	}

	return nil
}

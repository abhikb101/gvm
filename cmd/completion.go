package cmd

import (
	"github.com/gvm-tools/gvm/internal/profile"
	"github.com/spf13/cobra"
)

func init() {
	// Register dynamic profile name completion for all commands that take a profile name
	for _, cmd := range []*cobra.Command{
		useCmd, switchCmd, removeCmd, cloneCmd, loginCmd,
	} {
		cmd.ValidArgsFunction = completeProfileNames
	}
}

// completeProfileNames provides tab-completion for profile names.
func completeProfileNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only complete the first arg (profile name)
	if len(args) > 0 {
		// For clone, second arg is URL — no completion
		// For login, second arg is ssh/http
		if cmd.Name() == "login" && len(args) == 1 {
			return []string{"ssh", "http"}, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveDefault
	}

	profiles, err := profile.List()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var names []string
	for _, p := range profiles {
		names = append(names, p.Name+"\t"+p.GitEmail)
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

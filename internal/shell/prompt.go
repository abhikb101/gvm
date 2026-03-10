package shell

// StarshipConfig returns a TOML snippet for Starship prompt integration.
func StarshipConfig() string {
	return `[custom.gvm]
command = "cat ~/.gvm/active 2>/dev/null | tr -d '[:space:]'"
when = "test -f ~/.gvm/active"
format = "[$output]($style) "
style = "bold purple"
description = "Active GVM profile"`
}

// P10kSnippet returns instructions for Powerlevel10k integration.
func P10kSnippet() string {
	return `# Add gvm_prompt_info to your POWERLEVEL9K_LEFT_PROMPT_ELEMENTS or RIGHT_PROMPT_ELEMENTS:
# POWERLEVEL9K_RIGHT_PROMPT_ELEMENTS=(... gvm_profile)
#
# Define the segment:
# function prompt_gvm_profile() {
#   local profile=$(cat ~/.gvm/active 2>/dev/null | tr -d '[:space:]')
#   [[ -n "$profile" ]] && p10k segment -f purple -t "$profile" -i '🔑'
# }`
}

// OhMyZshSnippet returns instructions for Oh My Zsh integration.
func OhMyZshSnippet() string {
	return `# Add to your ~/.zshrc PROMPT or RPROMPT:
# RPROMPT='$(gvm_prompt_info) '$RPROMPT`
}

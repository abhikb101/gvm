package shell

import (
	"fmt"
	"os"
	"strings"

	"github.com/gvm-tools/gvm/internal/fsutil"
)

const hookMarkerStart = "# >>> gvm initialize >>>"
const hookMarkerEnd = "# <<< gvm initialize <<<"

// HookScript returns the auto-switch hook for the given shell.
func HookScript(s Shell) string {
	switch s {
	case Zsh:
		return zshHook()
	case Fish:
		return fishHook()
	default:
		return bashHook()
	}
}

// InstallHook appends the shell hook to the user's shell RC file.
// Returns true if the hook was installed, false if already present.
func InstallHook(s Shell) (bool, error) {
	rcPath := s.ConfigFile()

	// Ensure parent directory exists (for fish)
	if s == Fish {
		dir := rcPath[:strings.LastIndex(rcPath, "/")]
		if err := os.MkdirAll(dir, 0755); err != nil {
			return false, fmt.Errorf("creating config directory: %w", err)
		}
	}

	// Check if hook already installed
	content, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("reading %s: %w", rcPath, err)
	}
	if strings.Contains(string(content), hookMarkerStart) {
		return false, nil // already installed
	}

	// Append hook
	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, fmt.Errorf("opening %s: %w", rcPath, err)
	}
	defer f.Close()

	hookBlock := fmt.Sprintf("\n%s\n%s\n%s\n", hookMarkerStart, HookScript(s), hookMarkerEnd)
	if _, err := f.WriteString(hookBlock); err != nil {
		return false, fmt.Errorf("writing hook to %s: %w", rcPath, err)
	}

	return true, nil
}

// UninstallHook removes the GVM hook block from the shell RC file.
// Returns true if the hook was found and removed.
func UninstallHook(s Shell) (bool, error) {
	rcPath := s.ConfigFile()

	content, err := os.ReadFile(rcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("reading %s: %w", rcPath, err)
	}

	str := string(content)
	startIdx := strings.Index(str, "\n"+hookMarkerStart)
	if startIdx == -1 {
		startIdx = strings.Index(str, hookMarkerStart)
		if startIdx == -1 {
			return false, nil
		}
	}

	endMarker := hookMarkerEnd + "\n"
	endIdx := strings.Index(str, endMarker)
	if endIdx == -1 {
		endIdx = strings.Index(str, hookMarkerEnd)
		if endIdx == -1 {
			return false, nil
		}
		endIdx += len(hookMarkerEnd)
	} else {
		endIdx += len(endMarker)
	}

	newContent := str[:startIdx] + str[endIdx:]
	if err := fsutil.AtomicWrite(rcPath, []byte(newContent), 0644); err != nil {
		return false, fmt.Errorf("writing %s: %w", rcPath, err)
	}

	return true, nil
}

// IsHookInstalled checks if the GVM hook is in the shell RC file.
func IsHookInstalled(s Shell) bool {
	content, err := os.ReadFile(s.ConfigFile())
	if err != nil {
		return false
	}
	return strings.Contains(string(content), hookMarkerStart)
}

func zshHook() string {
	return `_gvm_find_gvmrc() {
  local dir="$PWD"
  while [ "$dir" != "/" ]; do
    if [ -f "$dir/.gvmrc" ]; then
      echo "$dir/.gvmrc"
      return 0
    fi
    dir=$(dirname "$dir")
  done
  return 1
}

gvm_auto_switch() {
  local gvmrc_path
  gvmrc_path=$(_gvm_find_gvmrc)
  if [ $? -eq 0 ]; then
    local profile=$(cat "$gvmrc_path" | tr -d '[:space:]')
    if [ -n "$profile" ]; then
      local current=$(cat ~/.gvm/active 2>/dev/null | tr -d '[:space:]')
      if [ "$profile" != "$current" ]; then
        gvm _activate "$profile" --quiet
      fi
    fi
  fi
}

gvm_prompt_info() {
  local profile=$(cat ~/.gvm/active 2>/dev/null | tr -d '[:space:]')
  if [ -n "$profile" ]; then
    echo "[gvm:$profile]"
  fi
}

autoload -U add-zsh-hook
add-zsh-hook chpwd gvm_auto_switch
gvm_auto_switch`
}

func bashHook() string {
	return `_gvm_find_gvmrc() {
  local dir="$PWD"
  while [ "$dir" != "/" ]; do
    if [ -f "$dir/.gvmrc" ]; then
      echo "$dir/.gvmrc"
      return 0
    fi
    dir=$(dirname "$dir")
  done
  return 1
}

gvm_auto_switch() {
  local gvmrc_path
  gvmrc_path=$(_gvm_find_gvmrc)
  if [ $? -eq 0 ]; then
    local profile=$(cat "$gvmrc_path" | tr -d '[:space:]')
    if [ -n "$profile" ]; then
      local current=$(cat ~/.gvm/active 2>/dev/null | tr -d '[:space:]')
      if [ "$profile" != "$current" ]; then
        gvm _activate "$profile" --quiet
      fi
    fi
  fi
}

gvm_prompt_info() {
  local profile=$(cat ~/.gvm/active 2>/dev/null | tr -d '[:space:]')
  if [ -n "$profile" ]; then
    echo "[gvm:$profile]"
  fi
}

PROMPT_COMMAND="gvm_auto_switch;${PROMPT_COMMAND}"
gvm_auto_switch`
}

func fishHook() string {
	return `function _gvm_find_gvmrc
  set -l dir $PWD
  while test "$dir" != "/"
    if test -f "$dir/.gvmrc"
      echo "$dir/.gvmrc"
      return 0
    end
    set dir (dirname "$dir")
  end
  return 1
end

function gvm_auto_switch --on-variable PWD
  set -l gvmrc_path (_gvm_find_gvmrc)
  if test $status -eq 0
    set -l profile (cat "$gvmrc_path" | string trim)
    if test -n "$profile"
      set -l current (cat ~/.gvm/active 2>/dev/null | string trim)
      if test "$profile" != "$current"
        gvm _activate "$profile" --quiet
      end
    end
  end
end

function gvm_prompt_info
  set -l profile (cat ~/.gvm/active 2>/dev/null | string trim)
  if test -n "$profile"
    echo "[gvm:$profile]"
  end
end

gvm_auto_switch`
}

<div align="center">

# gvm

**nvm for Git identities.** Switch between multiple GitHub accounts with one command.

[![CI](https://github.com/gvm-tools/gvm/actions/workflows/ci.yml/badge.svg)](https://github.com/gvm-tools/gvm/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/gvm-tools/gvm)](https://github.com/gvm-tools/gvm/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[Installation](#installation) · [Quick Start](#quick-start) · [Commands](#commands) · [How It Works](#how-it-works)

</div>

---

## The Problem

You have a personal GitHub account and a work account. Every time you clone a repo, you have to remember which SSH key to use. Your commits show up with the wrong email. You Google "multiple GitHub accounts" for the 47th time.

## The Solution

```bash
# Set up your identities once
gvm add personal
gvm add work

# Bind repos to identities
cd ~/my-side-project && gvm use personal
cd ~/company-repo && gvm use work

# That's it. GVM auto-switches when you cd between repos.
```

<!-- GIF DEMO HERE — use `vhs demo.tape` to record -->

## Installation

### Homebrew (macOS/Linux)

```bash
brew install gvm-tools/gvm/gvm
```

### Script

```bash
curl -sSL https://raw.githubusercontent.com/gvm-tools/gvm/main/scripts/install.sh | sh
```

### From source

```bash
go install github.com/gvm-tools/gvm@latest
```

## Quick Start

```bash
# 1. Initialize GVM and create your first profile
gvm init

# Already have SSH keys and git configs? Import them:
gvm migrate

# 2. Add another profile
gvm add work

# 3. Bind repos to profiles
cd ~/work-project
gvm use work

cd ~/personal-project
gvm use personal

# 4. GVM auto-switches as you navigate between repos
# Your commits, pushes, and pulls always use the right identity
```

## Commands

| Command | Description |
|---------|-------------|
| `gvm init` | Interactive first-time setup |
| `gvm add <name>` | Create a new identity profile |
| `gvm login <name> <ssh\|http>` | Add/update auth for a profile |
| `gvm use <name>` | Bind current repo to a profile |
| `gvm switch <name>` | Switch identity globally (session) |
| `gvm list` | Show all profiles |
| `gvm whoami` | Show active identity |
| `gvm clone <name> <url>` | Clone with specific identity |
| `gvm remove <name>` | Delete a profile |
| `gvm unbind` | Remove profile binding from current repo |
| `gvm migrate` | Import existing SSH keys, git configs, gh CLI auth |
| `gvm doctor` | Health check |
| `gvm config` | View/edit settings |

### `use` vs `switch`

- **`gvm use work`** — "This repo belongs to my work identity" (permanent, saved in `.gvmrc`)
- **`gvm switch work`** — "I want to be my work identity right now" (temporary, session-only)

Repo bindings (`use`) always override global switches.

## How It Works

GVM manages three things:

1. **SSH keys** — Each profile gets its own Ed25519 key (`~/.ssh/gvm_<name>`)
2. **Git config** — Automatically sets `user.name`, `user.email`, and `core.sshCommand` per-repo
3. **Shell hook** — Detects `.gvmrc` files on `cd` and auto-activates the right profile

No magic, no daemons, no background processes. Just config files and a shell hook.

## Shell Integration

GVM adds a small hook to your shell that auto-switches profiles when you `cd` into a bound repo. It also optionally shows the active profile in your prompt.

Works with: **zsh**, **bash**, **fish**

Prompt integration with: **Starship**, **Oh My Zsh**, **Powerlevel10k**

## Migrating Existing Setup

Already have multiple SSH keys and git configs? GVM can detect and import them:

```bash
gvm migrate --dry-run  # see what GVM finds (no changes)
gvm migrate            # interactively import existing identities
```

GVM scans for:
- Global git config (`user.name`, `user.email`)
- SSH keys in `~/.ssh/`
- Host entries in `~/.ssh/config` (the common `Host github-work` pattern)
- GitHub CLI (`gh`) authentication tokens
- `includeIf` directory-based git configs

## FAQ

**Does GVM modify my existing SSH config?**
No. GVM creates its own keys (prefixed with `gvm_`) and never touches your existing SSH setup.

**Can I use both SSH and HTTPS for the same profile?**
Yes. Run `gvm login <name> ssh` and `gvm login <name> http` to set up both.

**What if I forget to `gvm use` in a repo?**
Your global identity (set via `gvm switch`) will be used. Run `gvm whoami` to check.

**Does it work with GitLab/Bitbucket?**
Currently GitHub-only. GitLab and Bitbucket support is planned.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT — see [LICENSE](LICENSE)

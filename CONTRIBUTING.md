# Contributing to GVM

Thanks for your interest in contributing!

## Development Setup

```bash
git clone https://github.com/abhikb101/gvm.git
cd gvm
go mod download
go build -o gvm .
```

## Running Tests

```bash
go test -v -race ./...
```

## Code Style

- Run `gofmt` and `goimports` before committing
- All exported functions need doc comments
- Wrap errors with context: `fmt.Errorf("doing X: %w", err)`
- Write table-driven tests
- No global mutable state — pass dependencies explicitly

## Pull Requests

1. Fork the repo
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Commit with conventional commits (`feat:`, `fix:`, `docs:`, etc.)
4. Push and open a PR
5. Ensure CI passes

## Project Structure

```
cmd/          Command definitions (one file per command)
internal/     Internal packages (not importable externally)
  profile/    Profile CRUD and validation
  auth/       SSH key generation, OAuth device flow
  git/        Git config management, repo detection
  shell/      Shell hook generation, detection
  config/     Global config management
  crypto/     Token encryption/decryption
  ui/         Terminal output helpers (color, spinner, table)
  platform/   OS-specific operations (clipboard, browser, keychain)
```

## Reporting Issues

Use the issue templates. Include output of `gvm --version` and `gvm doctor`.

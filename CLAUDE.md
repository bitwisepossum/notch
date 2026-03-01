# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`notch` is a TUI-based TODO application written in Go (1.25). The UI layer is TUI-only for now, with potential future migration to a GUI via [Wails](https://wails.io). Keep UI logic decoupled from core logic to ease that transition.

## Storage Format

Todo lists are stored as Markdown files. Each file is one list, with the filename as the list name. Platform-native data directories:

- Linux: `~/.local/share/notch/`
- macOS: `~/Library/Application Support/notch/`
- Windows: `%APPDATA%\notch\`

Todos use GFM checkbox syntax. Hierarchy is expressed via 2-space indentation:

```md
# Work

- [ ] Project Alpha
  - [ ] Design phase
    - [x] Wireframes
    - [ ] Mockups
  - [ ] Development
    - [ ] Backend API
- [x] Team standup
```

The parser is hand-rolled (line-by-line, indent level = depth). No external Markdown library. Files remain valid Markdown and are editable outside the program.

## Common Commands

```sh
go build ./...       # build all packages
go run .             # run the TUI
go test ./...        # run all tests
go test ./pkg/...    # run tests in a specific package
go vet ./...         # lint
```

## Commit Style

- Short, imperative subject line (e.g. "Fix cursor reset on delete")
- No unnecessary detail — KISS
- No mentions of tooling, themes, or implementation flavor in the subject

## Version

The version string lives in `ui/version.go` — one constant, one file.

**Version push workflow:**
1. Update `Version` in `ui/version.go` (e.g. `"0.2.0"`)
2. Commit: `Bump version to v0.2.0`
3. Tag: `git tag v0.2.0`

Regular commits do not touch the version.

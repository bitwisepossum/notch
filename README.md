# notch

A TUI todo app. Lists are stored as Markdown files with GFM checkboxes, so they work in any editor.

![notch screenshot](images/example.png)

## Install

```sh
go install github.com/bitwisepossum/notch@latest
```

## Usage

```sh
notch
```

**List picker:** `j/k` / `↑/↓` move · `PgDn/PgUp` / `Shift+↑/↓` jump · `enter` open · `n` new · `d` delete · `q` quit

**Items:** `j/k` / `↑/↓` move · `PgDn/PgUp` / `Shift+↑/↓` jump · `space`/`enter` toggle · `a` add · `A` add child · `e` edit · `d` delete · `J/K` reorder · `tab`/`Shift+tab` indent/outdent · `esc` back · `q` quit

## Storage

Lists are saved as `.md` files in your platform's data directory:

- Linux: `~/.local/share/notch/`
- macOS: `~/Library/Application Support/notch/`
- Windows: `%APPDATA%\notch\`

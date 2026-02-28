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

### List picker

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Move cursor |
| `PgDn` / `PgUp` / `Shift+↑/↓` | Jump half page |
| `enter` | Open list |
| `n` | New list |
| `d` | Delete list |
| `q` | Quit |

### Items

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Move cursor |
| `PgDn` / `PgUp` / `Shift+↑/↓` | Jump half page |
| `space` / `enter` | Toggle done |
| `a` | Add item |
| `A` | Add child item |
| `e` | Edit item |
| `d` | Delete item |
| `J` / `K` / `Ctrl+↑/↓` | Reorder item |
| `tab` / `Shift+tab` | Indent / outdent |
| `esc` | Back to list picker |
| `q` | Quit |

## Storage

Lists are saved as `.md` files in your platform's data directory:

- Linux: `~/.local/share/notch/`
- macOS: `~/Library/Application Support/notch/`
- Windows: `%APPDATA%\notch\`

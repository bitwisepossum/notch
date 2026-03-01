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
| `s` | Settings |
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
| `/` | Search |
| `esc` | Back to list picker |
| `q` | Quit |

### Settings

| Key | Action |
|-----|--------|
| `j` / `k` | Move between settings |
| `e` | Set custom save path |
| `c` | Clear save path (revert to default) |
| `←` / `→` / `h` / `l` | Cycle theme |
| `esc` | Back |

## Storage

Lists are saved as `.md` files in your platform's data directory:

| Platform | Path |
|----------|------|
| Linux    | `~/.local/share/notch/` |
| macOS    | `~/Library/Application Support/notch/` |
| Windows  | `%APPDATA%\notch\` |

The save path can be changed from the Settings screen. The settings file itself (`settings.json`) always stays in the default location above.

## Themes

Themes are `.json` files placed in the `themes/` subfolder of the data directory. On first launch the folder is created automatically. See [`example-themes/`](example-themes/) for included examples and the full field reference.

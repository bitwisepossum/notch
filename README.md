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

Keybindings: `j/k` navigate, `enter/space` toggle, `a` add, `e` edit, `d` delete, `q` quit.

## Storage

Lists are saved as `.md` files in your platform's data directory:

- Linux: `~/.local/share/notch/`
- macOS: `~/Library/Application Support/notch/`
- Windows: `%APPDATA%\notch\`

# Example Themes

Copy the `.json` files you want into the `themes/` subdirectory of your notch data folder:

| Platform | Themes folder |
|----------|---------------|
| Linux    | `~/.local/share/notch/themes/` |
| macOS    | `~/Library/Application Support/notch/themes/` |
| Windows  | `%APPDATA%\notch\themes\` |

The folder is created automatically on first launch. Select a theme in **Settings → Theme** (`s` from the list view, then `←`/`→` to cycle).

## Included Examples

| File | Name | Description |
|------|------|-------------|
| `amber.json` | Amber | Warm amber CRT terminal palette |
| `dracula.json` | Dracula | [Dracula](https://draculatheme.com) dark theme |

## Creating Your Own

Copy any `.json` file and edit the hex color values. All fields are required:

```json
{
  "name":      "My Theme",
  "bg_select": "#hex",
  "muted":     "#hex",
  "primary":   "#hex",
  "accent":    "#hex",
  "danger":    "#hex",
  "separator": "#hex",
  "border":    "#hex",
  "done":      "#hex"
}
```

| Field | Used for |
|-------|----------|
| `bg_select` | Background of the selected row |
| `muted` | Dim text (help descriptions, counts, done checkboxes) |
| `primary` | Main item text, open checkboxes |
| `accent` | Title, cursor, highlights, keybindings |
| `danger` | Delete confirmations |
| `separator` | Depth dots, scrollbar track |
| `border` | Panel border |
| `done` | Strikethrough text on completed items |

The filename stem (e.g. `my-theme` from `my-theme.json`) is used as the internal key stored in `settings.json`. The `name` field is the display name shown in the UI.

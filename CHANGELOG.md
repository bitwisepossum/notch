# Changelog

## Unreleased

---

## v0.4.1 — 2026-03-18

### Added
- Log viewer accessible from settings; shows errors and events logged across the app, clear with a confirm prompt
- `LogLevel` setting: configurable verbosity, persisted in settings file
- Errors across storage, settings, and UI surfaced in the log: malformed checkboxes, theme load failures, silent save/load errors, active-theme fallback

---

## v0.4.0 — 2026-03-11

### Added
- F1 toggles a help sidebar in all views; panel width adjusts to make room
- Help sidebar compacts automatically when the terminal is too short: drops blank separators, then group headers, then folds into two columns
- Hide/show completed items with `H`
- Item count per list shown in the list picker
- Quick-add an item directly from the list picker (`a`)
- Undo/redo availability shown in the status bar (`u` / `^r` indicators)
- IO errors from failed saves surfaced in the status bar
- Long item text truncated with an ellipsis instead of overflowing

### Improved
- Status bar split: counts on the left, undo/redo and F1 hint right-aligned
- Straight border replaces thick border; frame and scrollbar track tightened
- Fold state preserved across move, indent, and outdent operations
- Cursor and scroll clamped on terminal resize
- Test coverage expanded for core logic and UI state transitions

### Fixed
- Help sidebar state (show/hide) not saved on quit
- F1 hint inconsistently shown or clipped across views

---

## v0.3.0 — 2026-03-04

### Added
- Deadline support: set/clear a deadline with `t`, badge display in item view
- Expiring, due soon, and due today status icons on deadline badges
- Configurable deadline display/input format (backend always stores YYYY-MM-DD)
- Progress bar in items header
- Status bar in items panel
- Thick border panel style
- Section and group headers in help sidebar
- Cascade done: marking a parent done can optionally mark all children done
- Themeable checkbox and progress bar characters
- Validation of theme files with errors surfaced in settings
- Centered popup overlay for confirm dialogs

---

## v0.2.5 — 2026-03-02

### Features
- Fold/unfold todo items via left/right arrow keys (context-aware: left collapses or jumps to parent, right expands)
- f key toggles fold on the current item
- Z folds all items with children / unfolds all
- Click the ▸/▾ indicator to toggle fold with the mouse
- Folded items show a (done/total) subtree count badge
### Improvements
- Search results show a muted parent breadcrumb to distinguish identically-named siblings across different parent items
- Folded items appear dimmed during search to indicate hidden children
- List header count always reflects the full list regardless of fold state
### Fixes
- Panel border jumping when folding (wide-character fold indicator replaced with narrow ▸)
---

## v0.2.0 — 2026-03-01

### Features
- Search (/) in the item browser
- Settings screen with custom save path and theme support
- Themes: live reload, filename display, custom themes directory
- List rename (r in list picker)
- Done/total counter in item view
- Mouse wheel scrolling and click support in settings
- Version string displayed in list picker
### UX
- Inline error when creating or renaming a list to an existing name
- q goes back in items/settings; quits only from the list picker
- Help sidebar improvements: merged alternate-key rows, removed duplicates
### Bug fixes
- List rename no longer panics or causes data loss on I/O failure
- Creating or renaming a list no longer silently overwrites an existing list
- New child cursor now follows the newly added item
- Move item no longer triggers a spurious save at sibling boundaries
### Security
- File reads/writes scoped with os.Root (Go 1.24) — fixes gosec G304/G306
- Settings file written with 0600 permissions — fixes gosec G306
### Build
- macOS binaries (amd64 + arm64) added to release artifacts

---

## v0.1.0 — 2026-02-28

Initial release.

### Features
- TUI-based TODO app with hierarchical items (2-space indent, unlimited depth)
- Markdown file storage, platform-native data directory
- Scrolling with overflow indicators and scrollbar
- Help key sidebar
- Page and half-page scroll keys
- Spacebar to toggle done state

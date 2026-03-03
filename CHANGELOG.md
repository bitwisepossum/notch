# Changelog

## v0.2.6

### Features
- Undo / redo (`u` / `Ctrl+r`) for all item mutations
- Fold state persists across sessions; stale state is discarded automatically when the list changes

### Improvements
- Cursor returns to the pre-search item when cancelling search with `esc`
- Stale fold state cleaned up on startup and when lists are deleted or renamed

### Fixes
- Unhandled file close errors in settings and theme loading

## v0.2.5

### Features
- Fold/unfold todo items via left/right arrow keys (context-aware: left collapses or jumps to parent, right expands)
- `f` key toggles fold on the current item
- `Z` folds all items with children / unfolds all
- Click the `▸`/`▾` indicator to toggle fold with the mouse
- Folded items show a `(done/total)` subtree count badge

### Improvements
- Search results show a muted parent breadcrumb (`H 0 › item`) to distinguish identically-named siblings
- Folded items appear dimmed during search to indicate hidden children
- List header count always reflects the full list regardless of fold state

### Fixes
- Panel border jumping when folding (wide-character fold indicator replaced with narrow `▸`)

## v0.2.0

- Initial public release


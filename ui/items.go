package ui

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/bitwisepossum/notch/todo"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// pathKey serializes a path slice to a string for use as a map key.
func pathKey(path []int) string {
	parts := make([]string, len(path))
	for i, v := range path {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

// flatItem is one row in the flattened tree view.
type flatItem struct {
	item  *todo.Item
	path  []int
	depth int
}

// rebuildFlat walks the item tree and produces a flat list for rendering.
// When a search query is active, only matching items are included.
func (m *Model) rebuildFlat() {
	m.ib.flat = m.ib.flat[:0]
	if m.ib.searchQuery != "" {
		for _, r := range m.ib.list.Search(m.ib.searchQuery) {
			if m.ib.hideDone && r.Item.Done {
				continue
			}
			m.ib.flat = append(m.ib.flat, flatItem{item: r.Item, path: r.Path, depth: len(r.Path) - 1})
		}
		return
	}
	m.flattenItems(m.ib.list.Items, nil, 0)
}

// flattenItems recursively appends items to m.ib.flat with their path and depth.
func (m *Model) flattenItems(items []*todo.Item, parentPath []int, depth int) {
	for i, item := range items {
		if m.ib.hideDone && item.Done {
			continue
		}
		path := make([]int, len(parentPath)+1)
		copy(path, parentPath)
		path[len(parentPath)] = i
		m.ib.flat = append(m.ib.flat, flatItem{item: item, path: path, depth: depth})
		if !m.ib.folded[pathKey(path)] {
			m.flattenItems(item.Children, path, depth+1)
		}
	}
}

// closeList saves state and returns to the list picker.
func (m *Model) closeList() {
	m.saveFlash()
	m.saveFoldState()
	m.persistShowHelp()
	m.ib.list = nil
	m.ib.flat = nil
	m.mode = modeListPicker
}

// updateItems handles messages while the item browser is active.
func (m Model) updateItems(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelDown:
			m.ib.cursor = min(m.ib.cursor+1, len(m.ib.flat)-1)
		case tea.MouseWheelUp:
			m.ib.cursor = max(m.ib.cursor-1, 0)
		}
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft && len(m.ib.flat) > 0 {
			idx := m.ib.scroll + (msg.Y - headerLines)
			if idx >= 0 && idx < len(m.ib.flat) {
				fi := m.ib.flat[idx]
				// frameLeft(2) + border(1) + innerPad(1) + scrollbar(0/1) + cursor(2) + dots(2*depth)
				scrollW := 0
				if len(m.ib.flat) > m.visibleRows() {
					scrollW = 1
				}
				foldX := 4 + scrollW + 2 + 2*fi.depth
				if msg.X >= foldX && msg.X < foldX+2 && len(fi.item.Children) > 0 {
					// Click on fold indicator → toggle fold
					key := pathKey(fi.path)
					if m.ib.folded[key] {
						delete(m.ib.folded, key)
					} else {
						m.ib.folded[key] = true
					}
					m.ib.cursor = idx
					m.rebuildFlat()
					m.ib.cursor = min(m.ib.cursor, max(len(m.ib.flat)-1, 0))
				} else if idx == m.ib.cursor {
					// Click on already-selected row → toggle done
					m.pushUndo()
					m.toggleDone(m.ib.flat[m.ib.cursor].path)
					m.saveFlash()
					m.rebuildFlat()
				} else {
					m.ib.cursor = idx
				}
			}
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q":
			m.closeList()
			return m, m.loadLists
		case "/":
			if len(m.ib.flat) > 0 {
				m.ib.preSearchItem = m.ib.flat[m.ib.cursor].item
			}
			m.mode = modeSearch
			m.input.textInput.SetValue(m.ib.searchQuery)
			return m, m.input.textInput.Focus()
		case "esc":
			if m.ib.searchQuery != "" {
				m.ib.searchQuery = ""
				m.rebuildFlat()
				if m.ib.preSearchItem != nil {
					m.followItem(m.ib.preSearchItem)
					m.ib.preSearchItem = nil
				} else {
					m.ib.cursor = min(m.ib.cursor, max(len(m.ib.flat)-1, 0))
				}
				m.ib.scroll = clampScroll(m.ib.cursor, m.ib.scroll, m.visibleRows(), len(m.ib.flat))
				return m, nil
			}
			m.closeList()
			return m, m.loadLists
		case "j", "down":
			if m.ib.cursor < len(m.ib.flat)-1 {
				m.ib.cursor++
			}
		case "k", "up":
			if m.ib.cursor > 0 {
				m.ib.cursor--
			}
		case "pgdown", "shift+down":
			m.ib.cursor = min(m.ib.cursor+m.halfPage(), len(m.ib.flat)-1)
		case "pgup", "shift+up":
			m.ib.cursor = max(m.ib.cursor-m.halfPage(), 0)
		case "u":
			m.undo()
			return m, nil
		case "ctrl+r":
			m.redo()
			return m, nil
		case "space", "enter":
			if len(m.ib.flat) > 0 {
				m.pushUndo()
				m.toggleDone(m.ib.flat[m.ib.cursor].path)
				m.saveFlash()
				m.rebuildFlat()
			}
		case "a":
			m.prevMode = modeItems
			m.mode = modeInput
			m.input.action = inputNewSibling
			m.input.textInput.SetValue("")
			return m, m.input.textInput.Focus()
		case "A":
			if len(m.ib.flat) > 0 {
				m.prevMode = modeItems
				m.mode = modeInput
				m.input.action = inputNewChild
				m.input.textInput.SetValue("")
				return m, m.input.textInput.Focus()
			}
		case "e":
			if len(m.ib.flat) > 0 {
				m.prevMode = modeItems
				m.mode = modeInput
				m.input.action = inputEditItem
				m.input.textInput.SetValue(m.ib.flat[m.ib.cursor].item.Text)
				return m, m.input.textInput.Focus()
			}
		case "d":
			if len(m.ib.flat) > 0 {
				fi := m.ib.flat[m.ib.cursor]
				m.prevMode = modeItems
				m.mode = modeConfirm
				m.confirm.msg = fmt.Sprintf("Delete %q?", fi.item.Text)
				m.confirm.kind = confirmDeleteItem
				m.confirm.itemPath = make([]int, len(fi.path))
				copy(m.confirm.itemPath, fi.path)
			}
		case "t":
			if len(m.ib.flat) > 0 {
				m.prevMode = modeItems
				m.mode = modeInput
				m.input.action = inputSetDeadline
				fi := m.ib.flat[m.ib.cursor]
				if !fi.item.Deadline.IsZero() {
					m.input.textInput.SetValue(fi.item.Deadline.Format(deadlineLayout(m.settings)))
				} else {
					m.input.textInput.SetValue("")
				}
				return m, m.input.textInput.Focus()
			}
		case "J", "ctrl+down":
			m.moveItem(1)
		case "K", "ctrl+up":
			m.moveItem(-1)
		case "tab":
			m.indentItem()
		case "shift+tab":
			m.outdentItem()
		case "left":
			m.foldItem()
		case "right":
			m.unfoldItem()
		case "f":
			m.toggleFold()
		case "H":
			m.ib.hideDone = !m.ib.hideDone
			m.rebuildFlat()
			m.ib.cursor = min(m.ib.cursor, max(len(m.ib.flat)-1, 0))
		case "Z":
			m.toggleFoldAll()
		}
	}
	m.ib.scroll = clampScroll(m.ib.cursor, m.ib.scroll, m.visibleRows(), len(m.ib.flat))
	return m, nil
}

// toggleDone toggles the done state of the item at path, cascading to children if enabled.
func (m *Model) toggleDone(path []int) {
	item := resolveItem(m.ib.list.Items, path)
	if m.settings.CascadeDone {
		_ = m.ib.list.ToggleCascade(path)
	} else {
		_ = m.ib.list.Toggle(path)
	}
	if item != nil {
		todo.LogEvent("item toggled", slog.String("list", m.ib.list.Name), slog.String("name", item.Text), slog.Bool("done", item.Done))
	}
}

// followItem updates the cursor to track a specific item after the flat list is rebuilt.
func (m *Model) followItem(target *todo.Item) {
	for i, f := range m.ib.flat {
		if f.item == target {
			m.ib.cursor = i
			return
		}
	}
}

// snapshotFoldedItems returns the set of item pointers that are currently folded.
// It walks the full item tree (not m.ib.flat) so folded items inside collapsed
// subtrees are also captured.
func (m *Model) snapshotFoldedItems() map[*todo.Item]bool {
	result := make(map[*todo.Item]bool)
	var walk func(items []*todo.Item, prefix []int)
	walk = func(items []*todo.Item, prefix []int) {
		for i, item := range items {
			path := make([]int, len(prefix)+1)
			copy(path, prefix)
			path[len(prefix)] = i
			if m.ib.folded[pathKey(path)] {
				result[item] = true
			}
			walk(item.Children, path)
		}
	}
	walk(m.ib.list.Items, nil)
	return result
}

// rebuildFoldedFromPointers replaces m.ib.folded with path keys derived from item
// pointer identity, preserving fold state across structural changes (move/indent/outdent).
func (m *Model) rebuildFoldedFromPointers(foldedItems map[*todo.Item]bool) {
	m.ib.folded = make(map[string]bool)
	var walk func(items []*todo.Item, prefix []int)
	walk = func(items []*todo.Item, prefix []int) {
		for i, item := range items {
			path := make([]int, len(prefix)+1)
			copy(path, prefix)
			path[len(prefix)] = i
			if foldedItems[item] {
				m.ib.folded[pathKey(path)] = true
			}
			walk(item.Children, path)
		}
	}
	walk(m.ib.list.Items, nil)
}

// moveItem swaps the current item with its sibling in the given direction (+1 down, -1 up).
func (m *Model) moveItem(dir int) {
	if len(m.ib.flat) == 0 {
		return
	}
	saved := m.snapshotFoldedItems()
	fi := m.ib.flat[m.ib.cursor]
	path := fi.path
	idx := path[len(path)-1]
	newIdx := idx + dir
	if newIdx < 0 {
		return
	}
	if newIdx >= m.ib.list.ChildCount(path[:len(path)-1]) {
		return
	}
	m.pushUndo()

	to := make([]int, len(path))
	copy(to, path)
	to[len(to)-1] = newIdx

	if err := m.ib.list.Move(path, to); err != nil {
		return
	}
	m.rebuildFoldedFromPointers(saved)
	m.saveFlash()
	m.rebuildFlat()
	m.followItem(fi.item)
}

// indentItem makes the current item a child of its previous sibling.
func (m *Model) indentItem() {
	if len(m.ib.flat) == 0 {
		return
	}
	saved := m.snapshotFoldedItems()
	fi := m.ib.flat[m.ib.cursor]
	path := fi.path
	idx := path[len(path)-1]
	if idx == 0 {
		return // no previous sibling to become child of
	}

	// New parent is the previous sibling at the same level.
	parentPath := make([]int, len(path))
	copy(parentPath, path)
	parentPath[len(parentPath)-1] = idx - 1

	// Navigate the tree to find the new parent and count its children.
	parent := resolveItem(m.ib.list.Items, parentPath)
	if parent == nil {
		return
	}
	m.pushUndo()
	childIdx := len(parent.Children) // append as last child

	// Build destination: parentPath + childIdx
	to := append(parentPath, childIdx)

	if err := m.ib.list.Move(path, to); err != nil {
		return
	}
	m.rebuildFoldedFromPointers(saved)
	m.saveFlash()
	m.rebuildFlat()
	m.followItem(fi.item)
}

// outdentItem moves the current item up one level (becomes a sibling of its parent).
func (m *Model) outdentItem() {
	if len(m.ib.flat) == 0 {
		return
	}
	saved := m.snapshotFoldedItems()
	fi := m.ib.flat[m.ib.cursor]
	path := fi.path
	if len(path) < 2 {
		return // already at top level
	}
	m.pushUndo()

	parentPath := path[:len(path)-1]
	parentIdx := parentPath[len(parentPath)-1]

	to := make([]int, len(parentPath))
	copy(to, parentPath)
	to[len(to)-1] = parentIdx + 1

	if err := m.ib.list.Move(path, to); err != nil {
		return
	}
	m.rebuildFoldedFromPointers(saved)
	m.saveFlash()
	m.rebuildFlat()
	m.followItem(fi.item)
}

// subtreeCount returns the total and done counts for all descendants of item.
func subtreeCount(item *todo.Item) (total, done int) {
	for _, c := range item.Children {
		total++
		if c.Done {
			done++
		}
		t, d := subtreeCount(c)
		total += t
		done += d
	}
	return
}

// foldItem collapses the current item if it has children and is expanded,
// or moves the cursor to its parent if it is already collapsed (or has no children).
func (m *Model) foldItem() {
	if len(m.ib.flat) == 0 {
		return
	}
	fi := m.ib.flat[m.ib.cursor]
	if len(fi.item.Children) > 0 && !m.ib.folded[pathKey(fi.path)] {
		m.ib.folded[pathKey(fi.path)] = true
		m.rebuildFlat()
		m.ib.cursor = min(m.ib.cursor, max(len(m.ib.flat)-1, 0))
	} else if fi.depth > 0 {
		parentKey := pathKey(fi.path[:len(fi.path)-1])
		for i, f := range m.ib.flat {
			if pathKey(f.path) == parentKey {
				m.ib.cursor = i
				break
			}
		}
	}
}

// unfoldItem expands the current item if it is collapsed.
func (m *Model) unfoldItem() {
	if len(m.ib.flat) == 0 {
		return
	}
	fi := m.ib.flat[m.ib.cursor]
	if m.ib.folded[pathKey(fi.path)] {
		delete(m.ib.folded, pathKey(fi.path))
		m.rebuildFlat()
	}
}

// toggleFold toggles the fold state of the current item if it has children.
func (m *Model) toggleFold() {
	if len(m.ib.flat) == 0 {
		return
	}
	fi := m.ib.flat[m.ib.cursor]
	if len(fi.item.Children) == 0 {
		return
	}
	key := pathKey(fi.path)
	if m.ib.folded[key] {
		delete(m.ib.folded, key)
	} else {
		m.ib.folded[key] = true
	}
	m.rebuildFlat()
	m.ib.cursor = min(m.ib.cursor, max(len(m.ib.flat)-1, 0))
}

// toggleFoldAll folds all items with children if nothing is folded, otherwise unfolds all.
func (m *Model) toggleFoldAll() {
	if len(m.ib.folded) > 0 {
		clear(m.ib.folded)
	} else {
		for _, fi := range m.ib.flat {
			if len(fi.item.Children) > 0 {
				m.ib.folded[pathKey(fi.path)] = true
			}
		}
	}
	m.rebuildFlat()
	m.ib.cursor = min(m.ib.cursor, max(len(m.ib.flat)-1, 0))
}

// truncateText shortens text so its display width fits within maxWidth, appending "…" when
// truncation occurs. Pass unstyled text; lipgloss.Width handles Unicode cell widths correctly.
func truncateText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= maxWidth {
		return text
	}
	runes := []rune(text)
	for i := len(runes); i > 0; i-- {
		candidate := string(runes[:i]) + "…"
		if lipgloss.Width(candidate) <= maxWidth {
			return candidate
		}
	}
	return "…"
}

// breadcrumb returns the › -joined names of ancestors along path (excluding the item itself).
func breadcrumb(items []*todo.Item, path []int) string {
	if len(path) < 2 {
		return ""
	}
	var parts []string
	slice := items
	for _, idx := range path[:len(path)-1] {
		if idx < 0 || idx >= len(slice) {
			break
		}
		parts = append(parts, slice[idx].Text)
		slice = slice[idx].Children
	}
	return strings.Join(parts, " › ")
}

// resolveItem navigates the item tree by path indices and returns the target item.
func resolveItem(items []*todo.Item, path []int) *todo.Item {
	var current *todo.Item
	slice := items
	for _, idx := range path {
		if idx < 0 || idx >= len(slice) {
			return nil
		}
		current = slice[idx]
		slice = current.Children
	}
	return current
}

// viewItems renders the item browser panel and help sidebar.
func (m Model) viewItems() string {
	var items strings.Builder

	total, done := subtreeCount(&todo.Item{Children: m.ib.list.Items})
	// Cache "today" once per render so we don't call time.Now() for each row.
	today := time.Now().In(time.Local).Truncate(24 * time.Hour)
	soonCutoff := today.Add(3 * 24 * time.Hour)

	if len(m.ib.flat) == 0 {
		if m.ib.hideDone && done > 0 {
			items.WriteString(styleEmpty.Render(fmt.Sprintf("All %d items done. Press H to show.", done)))
		} else {
			items.WriteString(styleEmpty.Render("Empty list. Press a to add an item."))
		}
	} else {
		visible := m.visibleRows()
		end := min(m.ib.scroll+visible, len(m.ib.flat))
		scrollWidth := 0
		if len(m.ib.flat) > visible {
			scrollWidth = 1
		}
		var lines []string
		for i := m.ib.scroll; i < end; i++ {
			lines = append(lines, m.renderItemRow(m.ib.flat[i], i == m.ib.cursor, today, soonCutoff, scrollWidth))
		}
		for len(lines) < visible {
			lines = append(lines, "")
		}
		if total := len(m.ib.flat); total > visible {
			si := computeScroll(m.ib.scroll, total, visible)
			lines = renderScrollbar(lines, si, m.panelWidth())
		}
		items.WriteString(strings.Join(lines, "\n"))
	}

	panel := stylePanel.Width(m.panelWidth()).Render(items.String())
	remaining := total - done
	var statusBar string
	if m.flashErr != "" {
		statusBar = styleConfirm.Render("  " + m.flashErr)
	} else {
		left := fmt.Sprintf("  %d done · %d remaining", done, remaining)
		if m.ib.hideDone && done > 0 {
			left += fmt.Sprintf(" · %d hidden", done)
		}
		leftStr := styleHelpDesc.Render(left)

		var rightParts []string
		if len(m.ib.undoStack) > 0 {
			rightParts = append(rightParts, styleHelpKey.Render("u"))
		}
		if len(m.ib.redoStack) > 0 {
			rightParts = append(rightParts, styleHelpKey.Render("^r"))
		}
		if hint := m.helpHint(); hint != "" {
			rightParts = append(rightParts, hint)
		}
		rightStr := strings.Join(rightParts, " ")
		statusBar = m.rightAlign(leftStr, rightStr)
	}
	title := styleTitle.Render(strings.ToUpper(m.ib.list.Name))
	bar := "  " + renderProgress(done, total, 14)
	count := "  " + styleCount.Render(fmt.Sprintf("%d/%d", done, total))
	if m.ib.searchQuery != "" {
		count += "  " + stylePrompt.Render("/") + styleHelpDesc.Render(m.ib.searchQuery)
	}
	// Items view: status bar sits below the panel, inside the help-join region.
	return m.layoutScreen(title+bar+count, panel+"\n"+statusBar, "")
}

// renderItemRow renders a single item row in the item browser.
func (m Model) renderItemRow(fi flatItem, selected bool, today, soonCutoff time.Time, scrollWidth int) string {
	key := pathKey(fi.path)
	isFolded := m.ib.folded[key]

	cursor := "  "
	if selected {
		cursor = styleCursor.Render("› ")
	}

	dots := ""
	if fi.depth > 0 {
		dots = styleDepthDot.Render(strings.Repeat("· ", fi.depth))
	}

	fold := "  "
	if len(fi.item.Children) > 0 {
		if m.ib.searchQuery != "" {
			if isFolded {
				fold = styleDepthDot.Render("▸") + " "
			}
		} else if isFolded {
			fold = styleCursor.Render("▸") + " "
		} else {
			fold = styleDepthDot.Render("▾") + " "
		}
	}

	check := styleCheckOpen.Render(charCheckOpen)
	if fi.item.Done {
		check = styleCheckDone.Render(charCheckDone)
	}

	suffix := ""
	if isFolded {
		t, d := subtreeCount(fi.item)
		suffix = " " + styleCount.Render(fmt.Sprintf("(%d/%d)", d, t))
	}
	if m.ib.hideDone && len(fi.item.Children) > 0 {
		hidden := 0
		for _, c := range fi.item.Children {
			if c.Done {
				hidden++
			}
		}
		if hidden > 0 {
			suffix += " " + styleCheckDone.Render(fmt.Sprintf("(+%d done)", hidden))
		}
	}

	deadlineBadge := ""
	if !fi.item.Deadline.IsZero() {
		dateStr := fi.item.Deadline.Format(deadlineLayout(m.settings))
		dl := fi.item.Deadline.In(time.Local).Truncate(24 * time.Hour)
		var ds lipgloss.Style
		icon := "#"
		switch {
		case fi.item.Done:
			ds = styleCheckDone
		case dl.Before(today):
			ds = styleConfirm
			icon = "!"
		case dl.Equal(today):
			ds = stylePrompt
			icon = "*"
		case dl.After(today) && !dl.After(soonCutoff):
			ds = styleHelpKey
			icon = "~"
		default:
			ds = styleHelpDesc
		}
		deadlineBadge = "  " + ds.Render(icon+" "+dateStr)
	}

	prefixWidth := 2 + 2*fi.depth + 2 + 1 + 2
	suffixWidth := lipgloss.Width(deadlineBadge) + lipgloss.Width(suffix)
	crumbWidth := 0
	if m.ib.searchQuery != "" && len(fi.path) > 1 {
		crumbWidth = lipgloss.Width(breadcrumb(m.ib.list.Items, fi.path) + " › ")
	}
	maxTextWidth := m.panelWidth() - 2 - scrollWidth - prefixWidth - suffixWidth - crumbWidth

	text := truncateText(fi.item.Text, maxTextWidth)
	if m.ib.searchQuery != "" && !fi.item.Done {
		text = highlightMatch(text, m.ib.searchQuery)
	}
	if fi.item.Done {
		text = styleDone.Render(text)
	} else if m.ib.searchQuery != "" && isFolded {
		text = styleCheckDone.Render(text)
	}
	if m.ib.searchQuery != "" && len(fi.path) > 1 {
		crumb := breadcrumb(m.ib.list.Items, fi.path)
		text = styleDepthDot.Render(crumb+" › ") + text
	}

	line := fmt.Sprintf("%s%s%s%s  %s%s%s", cursor, dots, fold, check, text, deadlineBadge, suffix)
	if selected {
		line = styleSelected.Render(line)
	}
	return line
}

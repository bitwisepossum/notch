package ui

import (
	"fmt"
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
	m.flat = m.flat[:0]
	if m.searchQuery != "" {
		for _, r := range m.list.Search(m.searchQuery) {
			if m.hideDone && r.Item.Done {
				continue
			}
			m.flat = append(m.flat, flatItem{item: r.Item, path: r.Path, depth: len(r.Path) - 1})
		}
		return
	}
	m.flattenItems(m.list.Items, nil, 0)
}

// flattenItems recursively appends items to m.flat with their path and depth.
func (m *Model) flattenItems(items []*todo.Item, parentPath []int, depth int) {
	for i, item := range items {
		if m.hideDone && item.Done {
			continue
		}
		path := make([]int, len(parentPath)+1)
		copy(path, parentPath)
		path[len(parentPath)] = i
		m.flat = append(m.flat, flatItem{item: item, path: path, depth: depth})
		if !m.folded[pathKey(path)] {
			m.flattenItems(item.Children, path, depth+1)
		}
	}
}

// updateItems handles messages while the item browser is active.
func (m Model) updateItems(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelDown:
			m.itemCursor = min(m.itemCursor+1, len(m.flat)-1)
		case tea.MouseWheelUp:
			m.itemCursor = max(m.itemCursor-1, 0)
		}
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft && len(m.flat) > 0 {
			idx := m.itemScroll + (msg.Y - headerLines)
			if idx >= 0 && idx < len(m.flat) {
				fi := m.flat[idx]
				// frameLeft(2) + border(1) + innerPad(1) + scrollbar(0/1) + cursor(2) + dots(2*depth)
				scrollW := 0
				if len(m.flat) > m.visibleRows() {
					scrollW = 1
				}
				foldX := 4 + scrollW + 2 + 2*fi.depth
				if msg.X >= foldX && msg.X < foldX+2 && len(fi.item.Children) > 0 {
					// Click on fold indicator → toggle fold
					key := pathKey(fi.path)
					if m.folded[key] {
						delete(m.folded, key)
					} else {
						m.folded[key] = true
					}
					m.itemCursor = idx
					m.rebuildFlat()
					m.itemCursor = min(m.itemCursor, max(len(m.flat)-1, 0))
				} else if idx == m.itemCursor {
					// Click on already-selected row → toggle done
					m.pushUndo()
					m.toggleDone(m.flat[m.itemCursor].path)
					m.save()
					m.rebuildFlat()
				} else {
					m.itemCursor = idx
				}
			}
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q":
			m.save()
			m.saveFoldState()
			m.list = nil
			m.flat = nil
			m.mode = modeListPicker
			return m, m.loadLists
		case "/":
			if len(m.flat) > 0 {
				m.preSearchItem = m.flat[m.itemCursor].item
			}
			m.mode = modeSearch
			m.textInput.SetValue(m.searchQuery)
			return m, m.textInput.Focus()
		case "esc":
			if m.searchQuery != "" {
				m.searchQuery = ""
				m.rebuildFlat()
				if m.preSearchItem != nil {
					m.followItem(m.preSearchItem)
					m.preSearchItem = nil
				} else {
					m.itemCursor = min(m.itemCursor, max(len(m.flat)-1, 0))
				}
				m.itemScroll = clampScroll(m.itemCursor, m.itemScroll, m.visibleRows(), len(m.flat))
				return m, nil
			}
			m.save()
			m.saveFoldState()
			m.list = nil
			m.flat = nil
			m.mode = modeListPicker
			return m, m.loadLists
		case "j", "down":
			if m.itemCursor < len(m.flat)-1 {
				m.itemCursor++
			}
		case "k", "up":
			if m.itemCursor > 0 {
				m.itemCursor--
			}
		case "pgdown", "shift+down":
			m.itemCursor = min(m.itemCursor+m.halfPage(), len(m.flat)-1)
		case "pgup", "shift+up":
			m.itemCursor = max(m.itemCursor-m.halfPage(), 0)
		case "u":
			m.undo()
			return m, nil
		case "ctrl+r":
			m.redo()
			return m, nil
		case "space", "enter":
			if len(m.flat) > 0 {
				m.pushUndo()
				m.toggleDone(m.flat[m.itemCursor].path)
				m.save()
				m.rebuildFlat()
			}
		case "a":
			m.prevMode = modeItems
			m.mode = modeInput
			m.inputAction = inputNewSibling
			m.textInput.SetValue("")
			return m, m.textInput.Focus()
		case "A":
			if len(m.flat) > 0 {
				m.prevMode = modeItems
				m.mode = modeInput
				m.inputAction = inputNewChild
				m.textInput.SetValue("")
				return m, m.textInput.Focus()
			}
		case "e":
			if len(m.flat) > 0 {
				m.prevMode = modeItems
				m.mode = modeInput
				m.inputAction = inputEditItem
				m.textInput.SetValue(m.flat[m.itemCursor].item.Text)
				return m, m.textInput.Focus()
			}
		case "d":
			if len(m.flat) > 0 {
				fi := m.flat[m.itemCursor]
				m.prevMode = modeItems
				m.mode = modeConfirm
				m.confirmMsg = fmt.Sprintf("Delete %q?", fi.item.Text)
				m.confirmKind = confirmDeleteItem
				m.confirmItemPath = make([]int, len(fi.path))
				copy(m.confirmItemPath, fi.path)
			}
		case "t":
			if len(m.flat) > 0 {
				m.prevMode = modeItems
				m.mode = modeInput
				m.inputAction = inputSetDeadline
				fi := m.flat[m.itemCursor]
				if !fi.item.Deadline.IsZero() {
					m.textInput.SetValue(fi.item.Deadline.Format(deadlineLayout(m.settings)))
				} else {
					m.textInput.SetValue("")
				}
				return m, m.textInput.Focus()
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
			m.hideDone = !m.hideDone
			m.rebuildFlat()
			m.itemCursor = min(m.itemCursor, max(len(m.flat)-1, 0))
		case "Z":
			m.toggleFoldAll()
		}
	}
	m.itemScroll = clampScroll(m.itemCursor, m.itemScroll, m.visibleRows(), len(m.flat))
	return m, nil
}

// toggleDone toggles the done state of the item at path, cascading to children if enabled.
func (m *Model) toggleDone(path []int) {
	if m.settings.CascadeDone {
		_ = m.list.ToggleCascade(path)
	} else {
		_ = m.list.Toggle(path)
	}
}

// followItem updates the cursor to track a specific item after the flat list is rebuilt.
func (m *Model) followItem(target *todo.Item) {
	for i, f := range m.flat {
		if f.item == target {
			m.itemCursor = i
			return
		}
	}
}

// snapshotFoldedItems returns the set of item pointers that are currently folded.
// It walks the full item tree (not m.flat) so folded items inside collapsed
// subtrees are also captured.
func (m *Model) snapshotFoldedItems() map[*todo.Item]bool {
	result := make(map[*todo.Item]bool)
	var walk func(items []*todo.Item, prefix []int)
	walk = func(items []*todo.Item, prefix []int) {
		for i, item := range items {
			path := make([]int, len(prefix)+1)
			copy(path, prefix)
			path[len(prefix)] = i
			if m.folded[pathKey(path)] {
				result[item] = true
			}
			walk(item.Children, path)
		}
	}
	walk(m.list.Items, nil)
	return result
}

// rebuildFoldedFromPointers replaces m.folded with path keys derived from item
// pointer identity, preserving fold state across structural changes (move/indent/outdent).
func (m *Model) rebuildFoldedFromPointers(foldedItems map[*todo.Item]bool) {
	m.folded = make(map[string]bool)
	var walk func(items []*todo.Item, prefix []int)
	walk = func(items []*todo.Item, prefix []int) {
		for i, item := range items {
			path := make([]int, len(prefix)+1)
			copy(path, prefix)
			path[len(prefix)] = i
			if foldedItems[item] {
				m.folded[pathKey(path)] = true
			}
			walk(item.Children, path)
		}
	}
	walk(m.list.Items, nil)
}

// moveItem swaps the current item with its sibling in the given direction (+1 down, -1 up).
func (m *Model) moveItem(dir int) {
	if len(m.flat) == 0 {
		return
	}
	saved := m.snapshotFoldedItems()
	fi := m.flat[m.itemCursor]
	path := fi.path
	idx := path[len(path)-1]
	newIdx := idx + dir
	if newIdx < 0 {
		return
	}
	if newIdx >= m.list.ChildCount(path[:len(path)-1]) {
		return
	}
	m.pushUndo()

	to := make([]int, len(path))
	copy(to, path)
	to[len(to)-1] = newIdx

	if err := m.list.Move(path, to); err != nil {
		return
	}
	m.rebuildFoldedFromPointers(saved)
	m.save()
	m.rebuildFlat()
	m.followItem(fi.item)
}

// indentItem makes the current item a child of its previous sibling.
func (m *Model) indentItem() {
	if len(m.flat) == 0 {
		return
	}
	saved := m.snapshotFoldedItems()
	fi := m.flat[m.itemCursor]
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
	parent := resolveItem(m.list.Items, parentPath)
	if parent == nil {
		return
	}
	m.pushUndo()
	childIdx := len(parent.Children) // append as last child

	// Build destination: parentPath + childIdx
	to := append(parentPath, childIdx)

	if err := m.list.Move(path, to); err != nil {
		return
	}
	m.rebuildFoldedFromPointers(saved)
	m.save()
	m.rebuildFlat()
	m.followItem(fi.item)
}

// outdentItem moves the current item up one level (becomes a sibling of its parent).
func (m *Model) outdentItem() {
	if len(m.flat) == 0 {
		return
	}
	saved := m.snapshotFoldedItems()
	fi := m.flat[m.itemCursor]
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

	if err := m.list.Move(path, to); err != nil {
		return
	}
	m.rebuildFoldedFromPointers(saved)
	m.save()
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
	if len(m.flat) == 0 {
		return
	}
	fi := m.flat[m.itemCursor]
	if len(fi.item.Children) > 0 && !m.folded[pathKey(fi.path)] {
		m.folded[pathKey(fi.path)] = true
		m.rebuildFlat()
		m.itemCursor = min(m.itemCursor, max(len(m.flat)-1, 0))
	} else if fi.depth > 0 {
		parentKey := pathKey(fi.path[:len(fi.path)-1])
		for i, f := range m.flat {
			if pathKey(f.path) == parentKey {
				m.itemCursor = i
				break
			}
		}
	}
}

// unfoldItem expands the current item if it is collapsed.
func (m *Model) unfoldItem() {
	if len(m.flat) == 0 {
		return
	}
	fi := m.flat[m.itemCursor]
	if m.folded[pathKey(fi.path)] {
		delete(m.folded, pathKey(fi.path))
		m.rebuildFlat()
	}
}

// toggleFold toggles the fold state of the current item if it has children.
func (m *Model) toggleFold() {
	if len(m.flat) == 0 {
		return
	}
	fi := m.flat[m.itemCursor]
	if len(fi.item.Children) == 0 {
		return
	}
	key := pathKey(fi.path)
	if m.folded[key] {
		delete(m.folded, key)
	} else {
		m.folded[key] = true
	}
	m.rebuildFlat()
	m.itemCursor = min(m.itemCursor, max(len(m.flat)-1, 0))
}

// toggleFoldAll folds all items with children if nothing is folded, otherwise unfolds all.
func (m *Model) toggleFoldAll() {
	if len(m.folded) > 0 {
		clear(m.folded)
	} else {
		for _, fi := range m.flat {
			if len(fi.item.Children) > 0 {
				m.folded[pathKey(fi.path)] = true
			}
		}
	}
	m.rebuildFlat()
	m.itemCursor = min(m.itemCursor, max(len(m.flat)-1, 0))
}

// resolveItem navigates the item tree by path indices and returns the target item.
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

	total, done := subtreeCount(&todo.Item{Children: m.list.Items})
	// Cache "today" once per render so we don't call time.Now() for each row.
	today := time.Now().In(time.Local).Truncate(24 * time.Hour)
	soonCutoff := today.Add(3 * 24 * time.Hour)

	if len(m.flat) == 0 {
		if m.hideDone && done > 0 {
			items.WriteString(styleEmpty.Render(fmt.Sprintf("All %d items done. Press H to show.", done)))
		} else {
			items.WriteString(styleEmpty.Render("Empty list. Press a to add an item."))
		}
	} else {
		visible := m.visibleRows()
		end := min(m.itemScroll+visible, len(m.flat))
		var lines []string
		for i := m.itemScroll; i < end; i++ {
			fi := m.flat[i]
			selected := i == m.itemCursor

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
				if m.searchQuery != "" {
					// During search folds are bypassed; show ▸ dimly if folded, nothing if expanded.
					if m.folded[pathKey(fi.path)] {
						fold = styleDepthDot.Render("▸") + " "
					}
				} else if m.folded[pathKey(fi.path)] {
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
			if m.folded[pathKey(fi.path)] {
				t, d := subtreeCount(fi.item)
				suffix = " " + styleCount.Render(fmt.Sprintf("(%d/%d)", d, t))
			}
			if m.hideDone && len(fi.item.Children) > 0 {
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
					// Due soon: gentle warning.
					ds = styleHelpKey
					icon = "~"
				default:
					ds = styleHelpDesc
				}
				deadlineBadge = "  " + ds.Render(icon+" "+dateStr)
			}

			isFolded := m.folded[pathKey(fi.path)]
			text := fi.item.Text
			if m.searchQuery != "" && !fi.item.Done {
				text = highlightMatch(text, m.searchQuery)
			}
			if fi.item.Done {
				text = styleDone.Render(text)
			} else if m.searchQuery != "" && isFolded {
				// During search, dim folded items so they stand out as having hidden children.
				text = styleCheckDone.Render(fi.item.Text)
			}
			if m.searchQuery != "" && len(fi.path) > 1 {
				crumb := breadcrumb(m.list.Items, fi.path)
				text = styleDepthDot.Render(crumb+" › ") + text
			}

			line := fmt.Sprintf("%s%s%s%s  %s%s%s", cursor, dots, fold, check, text, deadlineBadge, suffix)
			if selected {
				line = styleSelected.Render(line)
			}
			lines = append(lines, line)
		}
		// Pad to exactly `visible` rows so the panel height stays stable.
		for len(lines) < visible {
			lines = append(lines, "")
		}
		if total := len(m.flat); total > visible {
			si := computeScroll(m.itemScroll, total, visible)
			lines = renderScrollbar(lines, si, m.panelWidth())
		}
		items.WriteString(strings.Join(lines, "\n"))
	}

	panel := stylePanel.Width(m.panelWidth()).Render(items.String())
	remaining := total - done
	status := fmt.Sprintf("  %d done · %d remaining", done, remaining)
	if m.hideDone && done > 0 {
		status += fmt.Sprintf(" · %d hidden", done)
	}
	statusBar := styleHelpDesc.Render(status)
	help := lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(renderHelp(itemsHelp))

	var b strings.Builder
	title := styleTitle.Render(strings.ToUpper(m.list.Name))
	bar := "  " + renderProgress(done, total, 14)
	count := "  " + styleCount.Render(fmt.Sprintf("%d/%d", done, total))
	if m.searchQuery != "" {
		count += "  " + stylePrompt.Render("/") + styleHelpDesc.Render(m.searchQuery)
	}
	b.WriteString(title + bar + count + "\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, panel+"\n"+statusBar, help))
	return b.String()
}

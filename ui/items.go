package ui

import (
	"fmt"
	"strings"

	"github.com/bitwisepossum/notch/todo"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// flatItem is one row in the flattened tree view.
type flatItem struct {
	item  *todo.Item
	path  []int
	depth int
}

// rebuildFlat walks the item tree and produces a flat list for rendering.
func (m *Model) rebuildFlat() {
	m.flat = m.flat[:0]
	m.flattenItems(m.list.Items, nil, 0)
}

func (m *Model) flattenItems(items []*todo.Item, parentPath []int, depth int) {
	for i, item := range items {
		path := make([]int, len(parentPath)+1)
		copy(path, parentPath)
		path[len(parentPath)] = i
		m.flat = append(m.flat, flatItem{item: item, path: path, depth: depth})
		m.flattenItems(item.Children, path, depth+1)
	}
}

func (m Model) updateItems(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft && len(m.flat) > 0 {
			idx := m.itemScroll + (msg.Y - headerLines)
			if idx >= 0 && idx < len(m.flat) {
				if idx == m.itemCursor {
					// Click on already-selected row → toggle
					_ = m.list.Toggle(m.flat[m.itemCursor].path)
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
			return m, m.saveAndQuit
		case "esc":
			m.save()
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
		case "space", "enter":
			if len(m.flat) > 0 {
				_ = m.list.Toggle(m.flat[m.itemCursor].path)
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
				m.confirmMsg = fmt.Sprintf("Delete %q? (y/n)", fi.item.Text)
				m.confirmKind = confirmDeleteItem
				m.confirmItemPath = make([]int, len(fi.path))
				copy(m.confirmItemPath, fi.path)
			}
		case "J", "ctrl+down":
			m.moveItem(1)
		case "K", "ctrl+up":
			m.moveItem(-1)
		case "tab":
			m.indentItem()
		case "shift+tab":
			m.outdentItem()
		}
	}
	m.itemScroll = clampScroll(m.itemCursor, m.itemScroll, m.visibleRows(), len(m.flat))
	return m, nil
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

// moveItem swaps the current item with its sibling in the given direction (+1 down, -1 up).
func (m *Model) moveItem(dir int) {
	if len(m.flat) == 0 {
		return
	}
	fi := m.flat[m.itemCursor]
	path := fi.path
	idx := path[len(path)-1]
	newIdx := idx + dir
	if newIdx < 0 {
		return
	}

	to := make([]int, len(path))
	copy(to, path)
	to[len(to)-1] = newIdx

	if err := m.list.Move(path, to); err != nil {
		return
	}
	m.save()
	m.rebuildFlat()
	m.followItem(fi.item)
}

// indentItem makes the current item a child of its previous sibling.
func (m *Model) indentItem() {
	if len(m.flat) == 0 {
		return
	}
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
	childIdx := len(parent.Children) // append as last child

	// Build destination: parentPath + childIdx
	to := append(parentPath, childIdx)

	if err := m.list.Move(path, to); err != nil {
		return
	}
	m.save()
	m.rebuildFlat()
	m.followItem(fi.item)
}

// outdentItem moves the current item up one level (becomes a sibling of its parent).
func (m *Model) outdentItem() {
	if len(m.flat) == 0 {
		return
	}
	fi := m.flat[m.itemCursor]
	path := fi.path
	if len(path) < 2 {
		return // already at top level
	}

	parentPath := path[:len(path)-1]
	parentIdx := parentPath[len(parentPath)-1]

	to := make([]int, len(parentPath))
	copy(to, parentPath)
	to[len(to)-1] = parentIdx + 1

	if err := m.list.Move(path, to); err != nil {
		return
	}
	m.save()
	m.rebuildFlat()
	m.followItem(fi.item)
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

func (m Model) viewItems() string {
	var items strings.Builder

	if len(m.flat) == 0 {
		items.WriteString(styleEmpty.Render("Empty list. Press a to add an item."))
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

			check := styleCheckOpen.Render("[ ]")
			if fi.item.Done {
				check = styleCheckDone.Render("[x]")
			}

			text := fi.item.Text
			if fi.item.Done {
				text = styleDone.Render(text)
			}

			line := fmt.Sprintf("%s%s%s %s", cursor, dots, check, text)
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
	help := lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(renderHelp(itemsHelp))

	var b strings.Builder
	title := styleTitle.Render(strings.ToUpper(m.list.Name))
	count := styleCount.Render(fmt.Sprintf("  (%d)", len(m.flat)))
	b.WriteString(title + count + "\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, panel, help))
	return b.String()
}

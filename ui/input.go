package ui

import (
	"slices"
	"strings"

	"github.com/bitwisepossum/notch/todo"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// updateInput handles messages while the text input overlay is active.
func (m Model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(m.textInput.Value())
			m.inputErr = ""
			if val != "" {
				m.commitInput(val)
			}
			if m.inputErr == "" {
				m.textInput.Blur()
				m.mode = m.prevMode
			}
			return m, nil
		case "esc":
			m.inputErr = ""
			m.textInput.Blur()
			m.mode = m.prevMode
			return m, nil
		default:
			// User typed something — clear any standing error.
			m.inputErr = ""
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// commitInput applies the submitted text value based on the active inputAction.
func (m *Model) commitInput(val string) {
	switch m.inputAction {
	case inputNewList:
		if slices.Contains(m.lists, val) {
			m.inputErr = "a list named " + val + " already exists"
			return
		}
		// Create a new empty list file, then reload.
		newList := &todo.List{Name: val}
		_ = todo.Save(newList)
		m.lists, _ = todo.ListAll()
		// Move cursor to the new list.
		for i, name := range m.lists {
			if name == val {
				m.listCursor = i
				break
			}
		}

	case inputNewSibling:
		m.pushUndo()
		if len(m.flat) == 0 {
			// Empty list — add top-level item.
			m.list.Add(nil, val)
		} else {
			fi := m.flat[m.itemCursor]
			parentPath := fi.path[:len(fi.path)-1]
			// Insert after current item's index.
			siblingIdx := fi.path[len(fi.path)-1] + 1
			// Add at end then move into position.
			m.list.Add(parentPath, val)

			// The new item was appended at the end of parent's children.
			// Move it to the correct position if needed.
			parent := parentPath
			items := m.list.Items
			if len(parent) > 0 {
				p := resolveItem(items, parent)
				if p != nil {
					items = p.Children
				}
			}
			lastIdx := len(items) - 1
			if lastIdx != siblingIdx {
				from := make([]int, len(parent)+1)
				copy(from, parent)
				from[len(parent)] = lastIdx
				to := make([]int, len(parent)+1)
				copy(to, parent)
				to[len(parent)] = siblingIdx
				_ = m.list.Move(from, to)
			}
		}
		m.save()
		m.rebuildFlat()
		// Move cursor to the new item.
		m.itemCursor = min(m.itemCursor+1, len(m.flat)-1)

	case inputNewChild:
		if len(m.flat) > 0 {
			m.pushUndo()
			fi := m.flat[m.itemCursor]
			m.list.Add(fi.path, val)
			newChild := fi.item.Children[len(fi.item.Children)-1]
			m.save()
			m.rebuildFlat()
			m.followItem(newChild)
		}

	case inputEditItem:
		if len(m.flat) > 0 {
			m.pushUndo()
			fi := m.flat[m.itemCursor]
			_ = m.list.Edit(fi.path, val)
			m.save()
			m.rebuildFlat()
		}

	case inputSetDataDir:
		m.settings.CustomDataDir = val
		_ = todo.SaveSettings(m.settings)
		m.refreshListDir()
		m.lists, _ = todo.ListAll()
		m.listCursor = 0
		m.listScroll = 0

	case inputRenameList:
		oldName := m.lists[m.listCursor]
		if oldName != val && slices.Contains(m.lists, val) {
			m.inputErr = "a list named " + val + " already exists"
			return
		}
		if oldName != val {
			list, err := todo.Load(oldName)
			if err != nil {
				return
			}
			list.Rename(val)
			if err := todo.Save(list); err != nil {
				return
			}
			_ = todo.Delete(oldName)
			// Move any persisted fold state from old name to new name.
			s := m.settings
			if s.FoldState != nil {
				if entry, ok := s.FoldState[oldName]; ok {
					delete(s.FoldState, oldName)
					s.FoldState[val] = entry
					_ = todo.SaveSettings(s)
					m.settings = s
				}
			}
			m.lists, _ = todo.ListAll()
			for i, name := range m.lists {
				if name == val {
					m.listCursor = i
					break
				}
			}
		}
	}
}

// viewInput renders the underlying screen with the text input prompt below it.
func (m Model) viewInput() string {
	var b strings.Builder

	// Render the underlying screen.
	switch m.prevMode {
	case modeListPicker:
		b.WriteString(m.viewListPicker())
	case modeItems:
		b.WriteString(m.viewItems())
	case modeSettings:
		b.WriteString(m.viewSettings())
	}

	b.WriteString("\n")

	var prompt string
	switch m.inputAction {
	case inputNewList:
		prompt = "New list: "
	case inputNewSibling:
		prompt = "New item: "
	case inputNewChild:
		prompt = "New child: "
	case inputEditItem:
		prompt = "Edit: "
	case inputSetDataDir:
		prompt = "Save path: "
	case inputRenameList:
		prompt = "Rename: "
	}
	errStr := ""
	if m.inputErr != "" {
		errStr = "   " + styleConfirm.Render(m.inputErr)
	}
	b.WriteString(stylePrompt.Render(prompt) + m.textInput.View() + errStr + "\n")
	b.WriteString(renderHelp(inputHelp))
	return b.String()
}

// updateConfirm handles y/n key presses in the confirmation dialog.
func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "y":
			switch m.confirmKind {
			case confirmDeleteList:
				_ = todo.Delete(m.confirmTarget)
				// Remove any persisted fold state for the deleted list.
				s := m.settings
				if s.FoldState != nil {
					if _, ok := s.FoldState[m.confirmTarget]; ok {
						delete(s.FoldState, m.confirmTarget)
						_ = todo.SaveSettings(s)
						m.settings = s
					}
				}
				m.lists, _ = todo.ListAll()
				if m.listCursor >= len(m.lists) && m.listCursor > 0 {
					m.listCursor = len(m.lists) - 1
				}
			case confirmDeleteItem:
				m.pushUndo()
				clear(m.folded)
				_ = m.list.Remove(m.confirmItemPath)
				m.save()
				m.rebuildFlat()
				if m.itemCursor >= len(m.flat) && m.itemCursor > 0 {
					m.itemCursor = len(m.flat) - 1
				}
			}
			m.mode = m.prevMode
			return m, nil
		case "n", "esc":
			m.mode = m.prevMode
			return m, nil
		}
	}
	return m, nil
}

// viewConfirm renders a centered popup dialog overlaid on the underlying screen.
func (m Model) viewConfirm() string {
	// Render background at terminal dimensions.
	var underlying string
	switch m.prevMode {
	case modeListPicker:
		underlying = m.viewListPicker()
	case modeItems:
		underlying = m.viewItems()
	}
	bg := lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top,
		styleFrame.Render(underlying))

	// Build popup.
	msg := styleConfirm.Render(m.confirmMsg)
	hint := styleHelpKey.Render("y") + styleHelpDesc.Render("  /  ") + styleHelpKey.Render("n")
	innerWidth := max(lipgloss.Width(msg), lipgloss.Width(hint))
	popup := stylePanel.Width(innerWidth).Render(msg + "\n\n" + hint)

	return overlayCenter(bg, popup, m.width, m.height)
}

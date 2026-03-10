package ui

import (
	"slices"
	"strings"
	"time"

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
			// inputSetDeadline allows empty val (clears the deadline).
			if val != "" || m.inputAction == inputSetDeadline {
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

	case inputQuickAdd:
		name := m.lists[m.listCursor]
		list, err := todo.Load(name)
		if err != nil {
			return
		}
		list.Add(nil, val)
		_ = todo.Save(list)

	case inputSetDeadline:
		if len(m.flat) == 0 {
			return
		}
		fi := m.flat[m.itemCursor]
		m.pushUndo()
		if val == "" {
			_ = m.list.SetDeadline(fi.path, time.Time{})
		} else {
			d, err := time.Parse(deadlineLayout(m.settings), val)
			if err != nil {
				m.inputErr = "use " + deadlineLabel(m.settings) + " (e.g. " + deadlineHint(m.settings) + ")"
				return
			}
			_ = m.list.SetDeadline(fi.path, d)
		}
		m.save()
		m.rebuildFlat()
	}
}

// viewInput renders the underlying screen with the text input prompt below it.
func (m Model) viewInput() string {
	var b strings.Builder

	// Render the underlying screen.
	var underlying string
	switch m.prevMode {
	case modeListPicker:
		underlying = m.viewListPicker()
	case modeItems:
		underlying = m.viewItems()
	case modeSettings:
		underlying = m.viewSettings()
	}

	// The underlying views generally try to fill the entire terminal height.
	// When we add an input row + help row, the bottom of the screen can get
	// clipped, making the input appear "missing". Trim the underlying render to
	// leave room for the input UI.
	reserved := 3 // blank line + input line + help line
	maxUnderlying := max(m.height-reserved, 1)
	lines := strings.Split(underlying, "\n")
	if len(lines) > maxUnderlying {
		lines = lines[:maxUnderlying]
	}
	b.WriteString(strings.Join(lines, "\n"))
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
	case inputSetDeadline:
		prompt = "Deadline (" + deadlineLabel(m.settings) + ", empty=clear): "
	case inputQuickAdd:
		prompt = "Add to " + m.lists[m.listCursor] + ": "
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

	// Compose layers: background + popup centered within the panel area.
	popupW := lipgloss.Width(popup)
	popupH := lipgloss.Height(popup)
	// The panel occupies frameLeft(2) + border(1) + panelWidth + border(1).
	// Center the popup within that region.
	panelArea := 2 + 2 + m.panelWidth()
	bgLayer := lipgloss.NewLayer(bg)
	fgLayer := lipgloss.NewLayer(popup).
		X((panelArea - popupW) / 2).
		Y((m.height - popupH) / 2).
		Z(1)

	return lipgloss.NewCompositor(bgLayer, fgLayer).Render()
}

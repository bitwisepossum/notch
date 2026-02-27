package ui

import (
	"strings"

	"notch/todo"

	tea "charm.land/bubbletea/v2"
)

func (m Model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(m.textInput.Value())
			if val != "" {
				m.commitInput(val)
			}
			m.textInput.Blur()
			m.mode = m.prevMode
			return m, nil
		case "esc":
			m.textInput.Blur()
			m.mode = m.prevMode
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *Model) commitInput(val string) {
	switch m.inputAction {
	case inputNewList:
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
			fi := m.flat[m.itemCursor]
			m.list.Add(fi.path, val)
			m.save()
			m.rebuildFlat()
			// Move cursor to the new child (it's right after the current item in flat list).
			m.itemCursor++
		}

	case inputEditItem:
		if len(m.flat) > 0 {
			fi := m.flat[m.itemCursor]
			_ = m.list.Edit(fi.path, val)
			m.save()
			m.rebuildFlat()
		}
	}
}

func (m Model) viewInput() string {
	var b strings.Builder

	// Render the underlying screen.
	switch m.prevMode {
	case modeListPicker:
		b.WriteString(m.viewListPicker())
	case modeItems:
		b.WriteString(m.viewItems())
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
	}
	b.WriteString(stylePrompt.Render(prompt) + m.textInput.View() + "\n")
	b.WriteString(renderHelp(inputHelp))
	return b.String()
}

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "y":
			switch m.confirmKind {
			case confirmDeleteList:
				_ = todo.Delete(m.confirmTarget)
				m.lists, _ = todo.ListAll()
				if m.listCursor >= len(m.lists) && m.listCursor > 0 {
					m.listCursor = len(m.lists) - 1
				}
			case confirmDeleteItem:
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

func (m Model) viewConfirm() string {
	var b strings.Builder

	switch m.prevMode {
	case modeListPicker:
		b.WriteString(m.viewListPicker())
	case modeItems:
		b.WriteString(m.viewItems())
	}

	b.WriteString("\n")
	b.WriteString(styleConfirm.Render(m.confirmMsg) + " " +
		styleHelpKey.Render("y") + styleHelpDesc.Render("/") + styleHelpKey.Render("n"))
	return b.String()
}

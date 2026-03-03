package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// updateListPicker handles messages while the list selection screen is active.
func (m Model) updateListPicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelDown:
			m.listCursor = min(m.listCursor+1, len(m.lists)-1)
		case tea.MouseWheelUp:
			m.listCursor = max(m.listCursor-1, 0)
		}
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft && len(m.lists) > 0 {
			idx := m.listScroll + (msg.Y - headerLines)
			if idx >= 0 && idx < len(m.lists) {
				if idx == m.listCursor {
					// Click on already-selected row → open
					return m, m.openList(m.lists[m.listCursor])
				}
				m.listCursor = idx
			}
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "j", "down":
			if m.listCursor < len(m.lists)-1 {
				m.listCursor++
			}
		case "k", "up":
			if m.listCursor > 0 {
				m.listCursor--
			}
		case "pgdown", "shift+down":
			m.listCursor = min(m.listCursor+m.halfPage(), len(m.lists)-1)
		case "pgup", "shift+up":
			m.listCursor = max(m.listCursor-m.halfPage(), 0)
		case "enter":
			if len(m.lists) > 0 {
				return m, m.openList(m.lists[m.listCursor])
			}
		case "s":
			m.mode = modeSettings
		case "r":
			if len(m.lists) > 0 {
				m.prevMode = modeListPicker
				m.mode = modeInput
				m.inputAction = inputRenameList
				m.textInput.SetValue(m.lists[m.listCursor])
				return m, m.textInput.Focus()
			}
		case "n":
			m.prevMode = modeListPicker
			m.mode = modeInput
			m.inputAction = inputNewList
			m.textInput.SetValue("")
			return m, m.textInput.Focus()
		case "d":
			if len(m.lists) > 0 {
				name := m.lists[m.listCursor]
				m.prevMode = modeListPicker
				m.mode = modeConfirm
				m.confirmMsg = fmt.Sprintf("Delete list %q?", name)
				m.confirmKind = confirmDeleteList
				m.confirmTarget = name
			}
		}
	}
	m.listScroll = clampScroll(m.listCursor, m.listScroll, m.visibleRows(), len(m.lists))
	return m, nil
}

// viewListPicker renders the list selection panel and help sidebar.
func (m Model) viewListPicker() string {
	var items strings.Builder

	if len(m.lists) == 0 {
		items.WriteString(styleEmpty.Render("No lists yet. Press n to create one."))
	} else {
		visible := m.visibleRows()
		end := min(m.listScroll+visible, len(m.lists))
		var lines []string
		for i := m.listScroll; i < end; i++ {
			name := m.lists[i]
			var line string
			if i == m.listCursor {
				line = styleCursor.Render("› ") + styleSelected.Render(name)
			} else {
				line = "  " + name
			}
			lines = append(lines, line)
		}
		// Pad to exactly `visible` rows so the panel height stays stable.
		for len(lines) < visible {
			lines = append(lines, "")
		}
		if total := len(m.lists); total > visible {
			si := computeScroll(m.listScroll, total, visible)
			lines = renderScrollbar(lines, si, m.panelWidth())
		}
		items.WriteString(strings.Join(lines, "\n"))
	}

	panel := stylePanel.Width(m.panelWidth()).Render(items.String())
	help := lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(renderHelp(listHelp))

	var b strings.Builder
	title := styleTitle.Render("NOTCH")
	ver := styleHelpDesc.Render(" v" + Version)
	count := styleCount.Render(fmt.Sprintf("  (%d)", len(m.lists)))
	b.WriteString(title + ver + count + "\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, panel, help))
	if m.activeListDir != "" {
		b.WriteString("\n" + styleHelpDesc.Render(m.activeListDir))
	}
	return b.String()
}

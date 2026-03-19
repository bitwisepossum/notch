package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/bitwisepossum/notch/todo"
)

// updateListPicker handles messages while the list selection screen is active.
func (m Model) updateListPicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelDown:
			m.lp.cursor = min(m.lp.cursor+1, len(m.lp.lists)-1)
		case tea.MouseWheelUp:
			m.lp.cursor = max(m.lp.cursor-1, 0)
		}
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft && len(m.lp.lists) > 0 {
			idx := m.lp.scroll + (msg.Y - headerLines)
			if idx >= 0 && idx < len(m.lp.lists) {
				if idx == m.lp.cursor {
					// Click on already-selected row → open
					return m, m.openList(m.lp.lists[m.lp.cursor].name)
				}
				m.lp.cursor = idx
			}
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q":
			return m, m.saveAndQuit
		case "j", "down":
			if m.lp.cursor < len(m.lp.lists)-1 {
				m.lp.cursor++
			}
		case "k", "up":
			if m.lp.cursor > 0 {
				m.lp.cursor--
			}
		case "pgdown", "shift+down":
			m.lp.cursor = min(m.lp.cursor+m.halfPage(), len(m.lp.lists)-1)
		case "pgup", "shift+up":
			m.lp.cursor = max(m.lp.cursor-m.halfPage(), 0)
		case "enter":
			if len(m.lp.lists) > 0 {
				return m, m.openList(m.lp.lists[m.lp.cursor].name)
			}
		case "s":
			m.mode = modeSettings
		case "r":
			if len(m.lp.lists) > 0 {
				m.prevMode = modeListPicker
				m.mode = modeInput
				m.input.action = inputRenameList
				m.input.textInput.SetValue(m.lp.lists[m.lp.cursor].name)
				return m, m.input.textInput.Focus()
			}
		case "n":
			m.prevMode = modeListPicker
			m.mode = modeInput
			m.input.action = inputNewList
			m.input.textInput.SetValue("")
			return m, m.input.textInput.Focus()
		case "a":
			if len(m.lp.lists) > 0 {
				m.prevMode = modeListPicker
				m.mode = modeInput
				m.input.action = inputQuickAdd
				m.input.textInput.SetValue("")
				return m, m.input.textInput.Focus()
			}
		case "d":
			if len(m.lp.lists) > 0 {
				name := m.lp.lists[m.lp.cursor].name
				m.prevMode = modeListPicker
				m.mode = modeConfirm
				m.confirm.msg = fmt.Sprintf("Delete list %q?", name)
				m.confirm.kind = confirmDeleteList
				m.confirm.target = name
			}
		}
	}
	m.lp.scroll = clampScroll(m.lp.cursor, m.lp.scroll, m.visibleRows(), len(m.lp.lists))
	return m, nil
}

// viewListPicker renders the list selection panel and help sidebar.
func (m Model) viewListPicker() string {
	var items strings.Builder

	if len(m.lp.lists) == 0 {
		items.WriteString(styleEmpty.Render("No lists yet. Press n to create one."))
	} else {
		visible := m.visibleRows()
		end := min(m.lp.scroll+visible, len(m.lp.lists))
		var lines []string
		for i := m.lp.scroll; i < end; i++ {
			e := m.lp.lists[i]
			count := ""
			if e.totalItems > 0 {
				count = " " + styleCount.Render(fmt.Sprintf("%d/%d", e.doneItems, e.totalItems))
			}
			var line string
			if i == m.lp.cursor {
				line = styleSelected.Render("  " + e.name + count)
			} else {
				line = "  " + e.name + count
			}
			lines = append(lines, line)
		}
		// Pad to exactly `visible` rows so the panel height stays stable.
		for len(lines) < visible {
			lines = append(lines, "")
		}
		if total := len(m.lp.lists); total > visible {
			si := computeScroll(m.lp.scroll, total, visible)
			lines = renderScrollbar(lines, si, m.panelWidth())
		}
		items.WriteString(strings.Join(lines, "\n"))
	}

	panel := stylePanel.Width(m.panelWidth()).Render(items.String())

	title := styleTitle.Render("NOTCH")
	ver := styleHelpDesc.Render(" v" + todo.Version)
	count := styleCount.Render(fmt.Sprintf("  (%d)", len(m.lp.lists)))
	titleLine := title + ver + count

	hint := m.helpHint()
	var statusBar string
	if m.flashErr != "" {
		statusBar = m.rightAlign(styleConfirm.Render(m.flashErr), hint)
	} else if m.activeListDir != "" {
		statusBar = m.rightAlign(styleHelpDesc.Render(m.activeListDir), hint)
	} else if hint != "" {
		statusBar = m.rightAlign("", hint)
	}
	return m.layoutScreen(titleLine, panel, statusBar)
}

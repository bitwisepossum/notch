package ui

import (
	"strings"

	"github.com/bitwisepossum/notch/todo"

	tea "charm.land/bubbletea/v2"
)

// openLogViewer loads log lines and switches to the log viewer mode.
func (m *Model) openLogViewer() {
	raw, _ := todo.ReadLog()
	if raw == "" {
		m.log.lines = nil
	} else {
		m.log.lines = strings.Split(strings.TrimRight(raw, "\n"), "\n")
	}
	m.log.cursor = max(len(m.log.lines)-m.visibleRows(), 0) // start at bottom
	m.prevMode = modeSettings
	m.mode = modeLogViewer
}

// updateLogViewer handles input while the log viewer is active.
func (m Model) updateLogViewer(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		visible := m.visibleRows()
		total := len(m.log.lines)
		switch msg.String() {
		case "esc", "q":
			m.mode = modeSettings
		case "j", "down":
			m.log.cursor = min(m.log.cursor+1, max(total-visible, 0))
		case "k", "up":
			m.log.cursor = max(m.log.cursor-1, 0)
		case "pgdown", "shift+down":
			m.log.cursor = min(m.log.cursor+visible, max(total-visible, 0))
		case "pgup", "shift+up":
			m.log.cursor = max(m.log.cursor-visible, 0)
		case "g":
			m.log.cursor = 0
		case "G":
			m.log.cursor = max(total-visible, 0)
		case "C":
			m.prevMode = modeLogViewer
			m.confirm.msg = "Clear log file?"
			m.confirm.kind = confirmClearLog
			m.mode = modeConfirm
		}
	}
	return m, nil
}

// viewLogViewer renders the log file viewer panel.
func (m Model) viewLogViewer() string {
	visible := m.visibleRows()
	total := len(m.log.lines)

	scrollWidth := 0
	if total > visible {
		scrollWidth = 1
	}
	maxLineWidth := m.panelWidth() - 2 - scrollWidth

	lines := make([]string, visible)
	for i := range lines {
		lineIdx := m.log.cursor + i
		if lineIdx < total {
			lines[i] = truncateText(m.log.lines[lineIdx], maxLineWidth)
		}
	}

	si := computeScroll(m.log.cursor, total, visible)
	lines = renderScrollbar(lines, si, m.panelWidth())

	panel := stylePanel.Width(m.panelWidth()).Render(strings.Join(lines, "\n"))

	hint := m.helpHint()
	var statusBar string
	if m.flashErr != "" {
		statusBar = m.rightAlign(styleConfirm.Render(m.flashErr), hint)
	} else if hint != "" {
		statusBar = m.rightAlign("", hint)
	}
	return m.layoutScreen(styleTitle.Render("LOG"), panel, statusBar)
}

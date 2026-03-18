package ui

import (
	"strings"

	"github.com/bitwisepossum/notch/todo"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// openLogViewer loads log lines and switches to the log viewer mode.
func (m *Model) openLogViewer() {
	raw, _ := todo.ReadLog()
	if raw == "" {
		m.logLines = nil
	} else {
		m.logLines = strings.Split(strings.TrimRight(raw, "\n"), "\n")
	}
	m.logCursor = max(len(m.logLines)-m.visibleRows(), 0) // start at bottom
	m.prevMode = modeSettings
	m.mode = modeLogViewer
}

// updateLogViewer handles input while the log viewer is active.
func (m Model) updateLogViewer(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		visible := m.visibleRows()
		total := len(m.logLines)
		switch msg.String() {
		case "esc", "q":
			m.mode = modeSettings
		case "j", "down":
			m.logCursor = min(m.logCursor+1, max(total-visible, 0))
		case "k", "up":
			m.logCursor = max(m.logCursor-1, 0)
		case "pgdown", "shift+down":
			m.logCursor = min(m.logCursor+visible, max(total-visible, 0))
		case "pgup", "shift+up":
			m.logCursor = max(m.logCursor-visible, 0)
		case "g":
			m.logCursor = 0
		case "G":
			m.logCursor = max(total-visible, 0)
		case "C":
			m.prevMode = modeLogViewer
			m.confirmMsg = "Clear log file?"
			m.confirmKind = confirmClearLog
			m.mode = modeConfirm
		}
	}
	return m, nil
}

// viewLogViewer renders the log file viewer panel.
func (m Model) viewLogViewer() string {
	visible := m.visibleRows()
	total := len(m.logLines)

	lines := make([]string, visible)
	for i := range lines {
		lineIdx := m.logCursor + i
		if lineIdx < total {
			lines[i] = m.logLines[lineIdx]
		}
	}

	si := computeScroll(m.logCursor, total, visible)
	lines = renderScrollbar(lines, si, m.panelWidth())

	panel := stylePanel.Width(m.panelWidth()).Render(strings.Join(lines, "\n"))

	var b strings.Builder
	b.WriteString(styleTitle.Render("LOG") + "\n")
	if m.showHelp {
		help := lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(renderHelp(logViewerHelp, m.helpHeight()))
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, panel, help))
	} else {
		b.WriteString(panel)
	}
	hint := m.helpHint()
	if m.flashErr != "" {
		b.WriteString("\n" + m.rightAlign(styleConfirm.Render(m.flashErr), hint))
	} else if hint != "" {
		b.WriteString("\n" + m.rightAlign("", hint))
	} else {
		b.WriteString("\n")
	}
	return b.String()
}

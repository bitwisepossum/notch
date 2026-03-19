package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// layoutScreen assembles the standard screen layout: title line, panel with
// optional help sidebar, and a status bar line.
func (m Model) layoutScreen(title, panel, statusBar string) string {
	var b strings.Builder
	b.WriteString(title + "\n")
	if m.showHelp {
		help := lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(
			renderHelp(m.activeHelpGroups(), m.helpHeight()))
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, panel, help))
	} else {
		b.WriteString(panel)
	}
	if statusBar != "" {
		b.WriteString("\n" + statusBar)
	}
	return b.String()
}

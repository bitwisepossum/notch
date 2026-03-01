package ui

import (
	"strings"

	"github.com/bitwisepossum/notch/todo"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// updateSettings handles messages while the settings screen is active.
func (m Model) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "q":
			m.mode = modeListPicker
		case "e":
			m.prevMode = modeSettings
			m.mode = modeInput
			m.inputAction = inputSetDataDir
			m.textInput.SetValue(m.settings.CustomDataDir)
			return m, m.textInput.Focus()
		case "c":
			m.settings.CustomDataDir = ""
			_ = todo.SaveSettings(m.settings)
			m.lists, _ = todo.ListAll()
			m.listCursor = 0
			m.listScroll = 0
		}
	}
	return m, nil
}

// viewSettings renders the settings panel and help sidebar.
func (m Model) viewSettings() string {
	path := m.settings.CustomDataDir
	if path == "" {
		path = styleHelpDesc.Render("(default OS path)")
	} else {
		path = styleSelected.Render(path)
	}

	row := styleCursor.Render("› ") + styleHelpKey.Render("Save path") + "  " + path

	visible := m.visibleRows()
	lines := []string{row}
	for len(lines) < visible {
		lines = append(lines, "")
	}

	panel := stylePanel.Width(m.panelWidth()).Render(strings.Join(lines, "\n"))
	help := lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(renderHelp(settingsHelp))

	var b strings.Builder
	b.WriteString(styleTitle.Render("SETTINGS") + "\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, panel, help))
	return b.String()
}

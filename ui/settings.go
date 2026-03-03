package ui

import (
	"fmt"
	"strings"

	"github.com/bitwisepossum/notch/todo"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	settingsRowPath    = 0
	settingsRowTheme   = 1
	settingsRowCascade = 2
	settingsRowCount   = 3
)

// updateSettings handles messages while the settings screen is active.
func (m Model) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			row := msg.Y - headerLines
			if row >= 0 && row < settingsRowCount {
				if row == m.settingsCursor {
					// Second click on selected row — activate it.
					switch m.settingsCursor {
					case settingsRowPath:
						m.prevMode = modeSettings
						m.mode = modeInput
						m.inputAction = inputSetDataDir
						m.textInput.SetValue(m.settings.CustomDataDir)
						return m, m.textInput.Focus()
					case settingsRowTheme:
						m.cycleTheme(1)
					case settingsRowCascade:
						m.settings.CascadeDone = !m.settings.CascadeDone
						_ = todo.SaveSettings(m.settings)
					}
				} else {
					m.settingsCursor = row
				}
			}
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			m.mode = modeListPicker
		case "q":
			m.mode = modeListPicker

		case "j", "down":
			m.settingsCursor = (m.settingsCursor + 1) % settingsRowCount
		case "k", "up":
			m.settingsCursor = (m.settingsCursor - 1 + settingsRowCount) % settingsRowCount

		case "enter", "e":
			switch m.settingsCursor {
			case settingsRowPath:
				m.prevMode = modeSettings
				m.mode = modeInput
				m.inputAction = inputSetDataDir
				m.textInput.SetValue(m.settings.CustomDataDir)
				return m, m.textInput.Focus()
			case settingsRowTheme:
				m.cycleTheme(1)
			case settingsRowCascade:
				m.settings.CascadeDone = !m.settings.CascadeDone
				_ = todo.SaveSettings(m.settings)
			}

		case "c":
			if m.settingsCursor == settingsRowPath {
				m.settings.CustomDataDir = ""
				_ = todo.SaveSettings(m.settings)
				m.refreshListDir()
				m.lists, _ = todo.ListAll()
				m.listCursor = 0
				m.listScroll = 0
			}

		case "h", "left":
			if m.settingsCursor == settingsRowTheme {
				m.cycleTheme(-1)
			}
		case "l", "right":
			if m.settingsCursor == settingsRowTheme {
				m.cycleTheme(1)
			}

		case "R":
			m.themes = LoadThemes()
			m.applyActiveTheme()
		}
	}
	return m, nil
}

// cycleTheme advances the active theme by delta (-1 or +1), applies and saves it.
func (m *Model) cycleTheme(delta int) {
	if len(m.themes) == 0 {
		return
	}
	idx := (m.activeThemeIdx() + delta + len(m.themes)) % len(m.themes)
	t := m.themes[idx]
	m.settings.ActiveTheme = t.Key
	_ = todo.SaveSettings(m.settings)
	if t.Error != "" {
		applyTheme(DefaultTheme)
	} else {
		applyTheme(t)
	}
}

// viewSettings renders the settings panel and help sidebar.
func (m Model) viewSettings() string {
	rows := []struct {
		label string
		value string
	}{
		{
			label: "Save path",
			value: func() string {
				suffix := ""
				if m.settings.CustomDataDir == "" {
					suffix = styleHelpDesc.Render("  (default)")
				}
				return m.activeListDir + suffix
			}(),
		},
		{
			label: "Theme",
			value: func() string {
				idx := m.activeThemeIdx()
				t := DefaultTheme
				if idx < len(m.themes) {
					t = m.themes[idx]
				}
				total := len(m.themes)
				pos := styleHelpDesc.Render(fmt.Sprintf("[%d/%d]", idx+1, total))
				if t.Error != "" {
					return styleConfirm.Render(t.Key+".json") + "  " + styleHelpDesc.Render(t.Error) + "  " + pos
				}
				file := ""
				if t.Key != "" {
					file = styleHelpDesc.Render("  (" + t.Key + ".json)")
				}
				return fmt.Sprintf("%s%s  %s", t.Name, file, pos)
			}(),
		},
		{
			label: "Cascade done",
			value: func() string {
				if m.settings.CascadeDone {
					return styleCheckOpen.Render("on")
				}
				return styleHelpDesc.Render("off")
			}(),
		},
	}

	visible := m.visibleRows()
	lines := make([]string, 0, visible)
	for i, row := range rows {
		prefix := "  "
		label := styleHelpDesc.Render(row.label)
		value := row.value
		if i == m.settingsCursor {
			prefix = styleCursor.Render("› ")
			label = styleHelpKey.Render(row.label)
			value = styleSelected.Render(row.value)
		}
		lines = append(lines, prefix+label+"  "+value)
	}
	for len(lines) < visible {
		lines = append(lines, "")
	}

	panel := stylePanel.Width(m.panelWidth()).Render(strings.Join(lines, "\n"))
	help := lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2).Render(renderHelp(settingsHelp))

	var b strings.Builder
	b.WriteString(styleTitle.Render("SETTINGS") + "\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, panel, help))
	if m.themesDir != "" {
		b.WriteString("\n" + styleHelpDesc.Render("themes: "+m.themesDir))
	}
	return b.String()
}

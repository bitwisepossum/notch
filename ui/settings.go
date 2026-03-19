package ui

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bitwisepossum/notch/todo"

	tea "charm.land/bubbletea/v2"
)

const (
	settingsRowPath           = 0
	settingsRowTheme          = 1
	settingsRowDeadlineFormat = 2
	settingsRowCascade        = 3
	settingsRowLogLevel       = 4
	settingsRowViewLog        = 5
	settingsRowCount          = 6
)

// activateSetting performs the action for the currently selected settings row.
// Returns a tea.Cmd if the action requires one (e.g. focusing a text input).
func (m *Model) activateSetting() tea.Cmd {
	switch m.settingsCursor {
	case settingsRowPath:
		m.prevMode = modeSettings
		m.mode = modeInput
		m.input.action = inputSetDataDir
		m.input.textInput.SetValue(m.settings.CustomDataDir)
		return m.input.textInput.Focus()
	case settingsRowTheme:
		m.cycleTheme(1)
	case settingsRowDeadlineFormat:
		m.cycleDeadlineFormat(1)
	case settingsRowCascade:
		m.settings.CascadeDone = !m.settings.CascadeDone
		m.setFlash(todo.SaveSettings(m.settings))
		todo.LogEvent("cascade done changed", slog.Bool("enabled", m.settings.CascadeDone))
	case settingsRowLogLevel:
		m.cycleLogLevel(1)
	case settingsRowViewLog:
		m.openLogViewer()
	}
	return nil
}

// updateSettings handles messages while the settings screen is active.
func (m Model) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			row := msg.Y - headerLines
			if row >= 0 && row < settingsRowCount {
				if row == m.settingsCursor {
					if cmd := m.activateSetting(); cmd != nil {
						return m, cmd
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
			if cmd := m.activateSetting(); cmd != nil {
				return m, cmd
			}

		case "c":
			if m.settingsCursor == settingsRowPath {
				m.settings.CustomDataDir = ""
				m.setFlash(todo.SaveSettings(m.settings))
				todo.InvalidateListDir()
				m.refreshListDir()
				m.lp.lists = loadListEntries()
				m.lp.cursor = 0
				m.lp.scroll = 0
			}

		case "h", "left":
			switch m.settingsCursor {
			case settingsRowTheme:
				m.cycleTheme(-1)
			case settingsRowDeadlineFormat:
				m.cycleDeadlineFormat(-1)
			case settingsRowLogLevel:
				m.cycleLogLevel(-1)
			}
		case "l", "right":
			switch m.settingsCursor {
			case settingsRowTheme:
				m.cycleTheme(1)
			case settingsRowDeadlineFormat:
				m.cycleDeadlineFormat(1)
			case settingsRowLogLevel:
				m.cycleLogLevel(1)
			}

		case "R":
			m.themes = LoadThemes()
			m.applyActiveTheme()
		}
	}
	return m, nil
}

// cycleDeadlineFormat advances the deadline format preset by delta (-1 or +1) and saves it.
func (m *Model) cycleDeadlineFormat(delta int) {
	idx := (deadlineFormatIdx(m.settings) + delta + len(deadlineFormats)) % len(deadlineFormats)
	m.settings.DeadlineFormat = deadlineFormats[idx].layout
	m.setFlash(todo.SaveSettings(m.settings))
	todo.LogEvent("deadline format changed", slog.String("format", deadlineFormats[idx].label))
}

var logLevels = []string{todo.LogOff, todo.LogMinimal, todo.LogFull}

// logLevelIdx returns the current index of settings.LogLevel in logLevels.
func logLevelIdx(s todo.Settings) int {
	for i, l := range logLevels {
		if l == s.LogLevel {
			return i
		}
	}
	return 0 // default to off
}

// cycleLogLevel advances the log level by delta (-1 or +1), saves it, and reconfigures the logger.
func (m *Model) cycleLogLevel(delta int) {
	idx := (logLevelIdx(m.settings) + delta + len(logLevels)) % len(logLevels)
	m.settings.LogLevel = logLevels[idx]
	m.setFlash(todo.SaveSettings(m.settings))
	todo.SetLogLevel(m.settings.LogLevel)
	todo.LogEvent("log level changed", slog.String("level", m.settings.LogLevel))
}

// cycleTheme advances the active theme by delta (-1 or +1), applies and saves it.
func (m *Model) cycleTheme(delta int) {
	if len(m.themes) == 0 {
		return
	}
	idx := (m.activeThemeIdx() + delta + len(m.themes)) % len(m.themes)
	t := m.themes[idx]
	m.settings.ActiveTheme = t.Key
	m.setFlash(todo.SaveSettings(m.settings))
	todo.LogEvent("theme changed", slog.String("theme", t.Key))
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
			label: "Deadline format",
			value: func() string {
				idx := deadlineFormatIdx(m.settings)
				total := len(deadlineFormats)
				pos := styleHelpDesc.Render(fmt.Sprintf("[%d/%d]", idx+1, total))
				return deadlineLabel(m.settings) + "  " + pos
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
		{
			label: "Log level",
			value: func() string {
				idx := logLevelIdx(m.settings)
				total := len(logLevels)
				pos := styleHelpDesc.Render(fmt.Sprintf("[%d/%d]", idx+1, total))
				label := logLevels[idx]
				if label == todo.LogOff || label == "" {
					label = styleHelpDesc.Render(todo.LogOff)
				}
				return label + "  " + pos
			}(),
		},
		{
			label: "View log",
			value: func() string {
				sz := todo.LogSize()
				if sz == 0 {
					return styleHelpDesc.Render("empty")
				}
				return fmt.Sprintf("%.1f KB", float64(sz)/1024)
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

	hint := m.helpHint()
	var statusBar string
	if m.flashErr != "" {
		statusBar = m.rightAlign(styleConfirm.Render(m.flashErr), hint)
	} else if m.themesDir != "" {
		statusBar = m.rightAlign(styleHelpDesc.Render("themes: "+m.themesDir), hint)
	} else if hint != "" {
		statusBar = m.rightAlign("", hint)
	}
	return m.layoutScreen(styleTitle.Render("SETTINGS"), panel, statusBar)
}

package ui

import "charm.land/lipgloss/v2"

// Nokia LCD-inspired monochrome green palette.
var (
	colorBgSelect  = lipgloss.Color("#1A2508")
	colorMuted     = lipgloss.Color("#556820")
	colorPrimary   = lipgloss.Color("#9BB030")
	colorAccent    = lipgloss.Color("#D0E040")
	colorDanger    = lipgloss.Color("#C86050")
	colorSeparator = lipgloss.Color("#405010")
	colorBorder    = lipgloss.Color("#708830")
	colorDone      = lipgloss.Color("#3A4818")
)

var (
	styleTitle     = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleCursor    = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleDone      = lipgloss.NewStyle().Strikethrough(true).Foreground(colorDone)
	styleConfirm   = lipgloss.NewStyle().Bold(true).Foreground(colorDanger)
	styleCheckDone = lipgloss.NewStyle().Foreground(colorMuted)
	styleCheckOpen = lipgloss.NewStyle().Foreground(colorPrimary)
	styleSelected  = lipgloss.NewStyle().Background(colorBgSelect)
	styleEmpty     = lipgloss.NewStyle().Italic(true).Foreground(colorMuted)
	stylePrompt    = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleSeparator = lipgloss.NewStyle().Foreground(colorSeparator)
	styleCount     = lipgloss.NewStyle().Foreground(colorMuted)
	styleHelpKey   = lipgloss.NewStyle().Foreground(colorAccent)
	styleHelpDesc  = lipgloss.NewStyle().Foreground(colorMuted)
	styleDepthDot  = lipgloss.NewStyle().Foreground(colorSeparator)
	stylePanel     = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)
	styleFrame = lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2)
)

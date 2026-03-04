package ui

import (
	"charm.land/lipgloss/v2"
)

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

// UI characters — overridden by the active theme.
var (
	charCheckDone = "✓"
	charCheckOpen = "○"
	charBarFilled = "━"
	charBarEmpty  = "─"
)

var (
	styleTitle       = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleCursor      = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleDone        = lipgloss.NewStyle().Strikethrough(true).Foreground(colorDone)
	styleConfirm     = lipgloss.NewStyle().Bold(true).Foreground(colorDanger)
	styleCheckDone   = lipgloss.NewStyle().Foreground(colorMuted)
	styleCheckOpen   = lipgloss.NewStyle().Foreground(colorPrimary)
	styleSelected    = lipgloss.NewStyle().Background(colorBgSelect).Bold(true).Foreground(colorAccent)
	styleEmpty       = lipgloss.NewStyle().Italic(true).Foreground(colorMuted)
	stylePrompt      = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleCount       = lipgloss.NewStyle().Foreground(colorMuted)
	styleHelpKey     = lipgloss.NewStyle().Foreground(colorAccent)
	styleHelpDesc    = lipgloss.NewStyle().Foreground(colorMuted)
	styleDepthDot    = lipgloss.NewStyle().Foreground(colorSeparator)
	styleScrollThumb = lipgloss.NewStyle().Foreground(colorMuted)
	styleScrollTrack = lipgloss.NewStyle().Foreground(colorSeparator)
	styleScrollArrow = lipgloss.NewStyle().Foreground(colorMuted)
	stylePanel       = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorder).
				Padding(0, 1)
	styleFrame = lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2)
)

// applyTheme reassigns all color and style vars to match t.
func applyTheme(t Theme) {
	colorBgSelect = lipgloss.Color(t.BgSelect)
	colorMuted = lipgloss.Color(t.Muted)
	colorPrimary = lipgloss.Color(t.Primary)
	colorAccent = lipgloss.Color(t.Accent)
	colorDanger = lipgloss.Color(t.Danger)
	colorSeparator = lipgloss.Color(t.Separator)
	colorBorder = lipgloss.Color(t.Border)
	colorDone = lipgloss.Color(t.Done)

	orDefault := func(s, def string) string {
		if s != "" {
			return s
		}
		return def
	}
	charCheckDone = orDefault(t.CheckDone, "✓")
	charCheckOpen = orDefault(t.CheckOpen, "○")
	charBarFilled = orDefault(t.BarFilled, "━")
	charBarEmpty = orDefault(t.BarEmpty, "─")

	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleCursor = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleDone = lipgloss.NewStyle().Strikethrough(true).Foreground(colorDone)
	styleConfirm = lipgloss.NewStyle().Bold(true).Foreground(colorDanger)
	styleCheckDone = lipgloss.NewStyle().Foreground(colorMuted)
	styleCheckOpen = lipgloss.NewStyle().Foreground(colorPrimary)
	styleSelected = lipgloss.NewStyle().Background(colorBgSelect).Bold(true).Foreground(colorAccent)
	styleEmpty = lipgloss.NewStyle().Italic(true).Foreground(colorMuted)
	stylePrompt = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleCount = lipgloss.NewStyle().Foreground(colorMuted)
	styleHelpKey = lipgloss.NewStyle().Foreground(colorAccent)
	styleHelpDesc = lipgloss.NewStyle().Foreground(colorMuted)
	styleDepthDot = lipgloss.NewStyle().Foreground(colorSeparator)
	styleScrollThumb = lipgloss.NewStyle().Foreground(colorMuted)
	styleScrollTrack = lipgloss.NewStyle().Foreground(colorSeparator)
	styleScrollArrow = lipgloss.NewStyle().Foreground(colorMuted)
	stylePanel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1)
	styleFrame = lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2)
}

package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// helpPair is a description–key pair for the help sidebar.
type helpPair struct {
	keys string
	desc string
}

// helpGroup is a named group of helpPairs. header is optional; empty means
// no header, just a blank-line separator between groups.
type helpGroup struct {
	header string
	pairs  []helpPair
}

// renderHelp builds a vertical help sidebar from a slice of helpGroups.
// Groups with a non-empty header show it in a muted style above their pairs.
// Groups are separated by a blank line.
func renderHelp(groups []helpGroup) string {
	// Compute max desc width across all pairs.
	maxDesc := 0
	for _, g := range groups {
		for _, p := range g.pairs {
			if w := lipgloss.Width(p.desc); w > maxDesc {
				maxDesc = w
			}
		}
	}

	var lines []string
	for i, g := range groups {
		if i > 0 {
			lines = append(lines, "")
		}
		if g.header != "" {
			lines = append(lines, styleCount.Bold(true).Render(g.header))
		}
		for _, p := range g.pairs {
			descPad := strings.Repeat(" ", maxDesc-lipgloss.Width(p.desc))
			line := styleHelpDesc.Render(p.desc+descPad) + "  " + styleHelpKey.Render(p.keys)
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

var (
	listHelp = []helpGroup{
		{pairs: []helpPair{
			{"j/k ↑/↓", "move"},
			{"PgDn/PgUp S-↑/↓", "jump"},
			{"enter", "open"},
		}},
		{pairs: []helpPair{
			{"n", "new"},
			{"r", "rename"},
			{"d", "delete"},
			{"s", "settings"},
			{"q", "quit"},
		}},
	}
	settingsHelp = []helpGroup{
		{pairs: []helpPair{
			{"j/k", "move"},
		}},
		{pairs: []helpPair{
			{"enter/e", "activate"},
			{"c", "clear path"},
		}},
		{pairs: []helpPair{
			{"←/→ h/l", "theme"},
			{"R", "reload themes"},
			{"esc/q", "back"},
		}},
	}
	itemsHelp = []helpGroup{
		{header: "navigate", pairs: []helpPair{
			{"j/k ↑/↓", "move"},
			{"PgDn/PgUp S-↑/↓", "jump"},
			{"←/→", "fold"},
			{"f", "toggle fold"},
			{"Z", "fold all"},
		}},
		{header: "edit", pairs: []helpPair{
			{"space/enter", "toggle"},
			{"a", "add"},
			{"A", "child"},
			{"e", "edit"},
			{"t", "deadline"},
			{"d", "delete"},
		}},
		{header: "move", pairs: []helpPair{
			{"J/K C-↑/↓", "reorder"},
			{"tab", "indent"},
			{"S-tab", "outdent"},
		}},
		{header: "search", pairs: []helpPair{
			{"/", "search"},
			{"esc", "back/clear"},
			{"q", "back"},
		}},
		{header: "history", pairs: []helpPair{
			{"u", "undo"},
			{"C-r", "redo"},
		}},
	}
	inputHelp = []helpGroup{
		{pairs: []helpPair{
			{"enter", "confirm"},
			{"esc", "cancel"},
		}},
	}
)

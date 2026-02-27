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

// renderHelp builds a vertical help sidebar: one pair per line, groups separated
// by a blank line. Format is desc (muted) then key (accent), columns aligned.
func renderHelp(groups [][]helpPair) string {
	// Compute max desc width across all pairs.
	maxDesc := 0
	for _, g := range groups {
		for _, p := range g {
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
		for _, p := range g {
			descPad := strings.Repeat(" ", maxDesc-lipgloss.Width(p.desc))
			line := styleHelpDesc.Render(p.desc+descPad) + "  " + styleHelpKey.Render(p.keys)
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

var (
	listHelp = [][]helpPair{
		{
			{"j/k ↑/↓", "move"},
			{"enter", "open"},
		},
		{
			{"n", "new"},
			{"d", "delete"},
			{"q", "quit"},
		},
	}
	itemsHelp = [][]helpPair{
		{
			{"j/k ↑/↓", "move"},
		},
		{
			{"space", "toggle"},
			{"a", "add"},
			{"A", "child"},
			{"e", "edit"},
			{"d", "delete"},
		},
		{
			{"J/K", "reorder"},
			{"tab", "indent"},
			{"S-tab", "outdent"},
		},
		{
			{"esc", "back"},
			{"q", "quit"},
		},
	}
	inputHelp = [][]helpPair{
		{
			{"enter", "confirm"},
			{"esc", "cancel"},
		},
	}
)

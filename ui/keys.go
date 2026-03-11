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
//
// maxLines limits the output height. When content exceeds maxLines it
// progressively compacts: first blank separators are dropped, then group
// headers, and only as a last resort are lines clipped. Pass 0 for no limit.
func renderHelp(groups []helpGroup, maxLines int) string {
	// Compute max desc width across all pairs.
	maxDesc := 0
	for _, g := range groups {
		for _, p := range g.pairs {
			if w := lipgloss.Width(p.desc); w > maxDesc {
				maxDesc = w
			}
		}
	}

	build := func(blanks, headers bool) []string {
		var lines []string
		for i, g := range groups {
			if i > 0 && blanks {
				lines = append(lines, "")
			}
			if headers && g.header != "" {
				lines = append(lines, styleCount.Bold(true).Render(g.header))
			}
			for _, p := range g.pairs {
				descPad := strings.Repeat(" ", maxDesc-lipgloss.Width(p.desc))
				line := styleHelpDesc.Render(p.desc+descPad) + "  " + styleHelpKey.Render(p.keys)
				lines = append(lines, line)
			}
		}
		return lines
	}

	lines := build(true, true)
	if maxLines > 0 && len(lines) > maxLines {
		lines = build(false, true)
	}
	if maxLines > 0 && len(lines) > maxLines {
		lines = build(false, false)
	}
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
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
			{"a", "quick add"},
			{"r", "rename"},
			{"d", "delete"},
			{"s", "settings"},
			{"q", "quit"},
		}},
		{pairs: []helpPair{
			{"F1", "hide help"},
		}},
	}
	settingsHelp = []helpGroup{
		{pairs: []helpPair{
			{"j/k ↑/↓", "move"},
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
		{pairs: []helpPair{
			{"F1", "hide help"},
		}},
	}
	itemsHelp = []helpGroup{
		{header: "navigate", pairs: []helpPair{
			{"j/k ↑/↓", "move"},
			{"PgDn/PgUp S-↑/↓", "jump"},
			{"←/→", "fold"},
			{"f", "toggle fold"},
			{"Z", "fold all"},
			{"H", "hide done"},
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
		{pairs: []helpPair{
			{"F1", "hide help"},
		}},
	}
	inputHelp = []helpGroup{
		{pairs: []helpPair{
			{"enter", "confirm"},
			{"esc", "cancel"},
		}},
	}
)

package ui

import "strings"

// helpPair is a key–description pair for the help bar.
type helpPair struct {
	keys string
	desc string
}

// renderHelp builds a styled help string from key–description pairs.
func renderHelp(pairs []helpPair) string {
	sep := styleSeparator.Render(" · ")
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = styleHelpKey.Render(p.keys) + styleHelpDesc.Render(": "+p.desc)
	}
	return strings.Join(parts, sep)
}

var (
	listHelp = []helpPair{
		{"j/k", "navigate"},
		{"enter", "open"},
		{"n", "new"},
		{"d", "delete"},
		{"q", "quit"},
	}
	itemsHelp = []helpPair{
		{"j/k", "navigate"},
		{"space", "toggle"},
		{"a", "add"},
		{"A", "add child"},
		{"e", "edit"},
		{"d", "delete"},
		{"J/K", "move"},
		{"tab/S-tab", "indent"},
		{"esc", "back"},
		{"q", "quit"},
	}
	inputHelp = []helpPair{
		{"enter", "confirm"},
		{"esc", "cancel"},
	}
)

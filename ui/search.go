package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// updateSearch handles messages while the search input overlay is active.
// Arrow keys navigate the filtered list; all other keys are forwarded to the
// text input so the user can type freely. Enter confirms, esc cancels.
func (m Model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			m.searchQuery = ""
			m.textInput.Blur()
			m.textInput.SetValue("")
			m.mode = modeItems
			m.rebuildFlat()
			if m.preSearchItem != nil {
				m.followItem(m.preSearchItem)
				m.preSearchItem = nil
			} else {
				m.itemCursor = min(m.itemCursor, max(len(m.flat)-1, 0))
			}
			m.itemScroll = clampScroll(m.itemCursor, m.itemScroll, m.visibleRows(), len(m.flat))
			return m, nil
		case "enter":
			m.textInput.Blur()
			m.mode = modeItems
			m.itemCursor = min(m.itemCursor, max(len(m.flat)-1, 0))
			m.itemScroll = clampScroll(m.itemCursor, m.itemScroll, m.visibleRows(), len(m.flat))
			return m, nil
		case "up":
			if m.itemCursor > 0 {
				m.itemCursor--
			}
			m.itemScroll = clampScroll(m.itemCursor, m.itemScroll, m.visibleRows(), len(m.flat))
			return m, nil
		case "down":
			if m.itemCursor < len(m.flat)-1 {
				m.itemCursor++
			}
			m.itemScroll = clampScroll(m.itemCursor, m.itemScroll, m.visibleRows(), len(m.flat))
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m.searchQuery = m.textInput.Value()
	m.rebuildFlat()
	m.itemCursor = min(m.itemCursor, max(len(m.flat)-1, 0))
	m.itemScroll = clampScroll(m.itemCursor, m.itemScroll, m.visibleRows(), len(m.flat))
	return m, cmd
}

// viewSearch renders the filtered item list with the search prompt below.
func (m Model) viewSearch() string {
	var b strings.Builder
	b.WriteString(m.viewItems())
	b.WriteString("\n")
	b.WriteString(stylePrompt.Render("/") + " " + m.textInput.View() + "\n")
	b.WriteString(renderHelp(searchHelp, 0))
	return b.String()
}

// highlightMatch wraps the first occurrence of query (case-insensitive) in
// the accent style, leaving the rest of text unstyled.
func highlightMatch(text, query string) string {
	if query == "" {
		return text
	}
	idx := strings.Index(strings.ToLower(text), strings.ToLower(query))
	if idx < 0 {
		return text
	}
	end := idx + len(query)
	return text[:idx] + styleHelpKey.Render(text[idx:end]) + text[end:]
}

var searchHelp = []helpGroup{
	{pairs: []helpPair{
		{"↑/↓", "navigate"},
		{"enter", "confirm"},
		{"esc", "cancel"},
	}},
}

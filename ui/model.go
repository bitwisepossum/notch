package ui

import (
	"notch/todo"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

type mode int

const (
	modeListPicker mode = iota
	modeItems
	modeInput
	modeConfirm
)

// inputAction tracks what the text input is being used for.
type inputAction int

const (
	inputNewList inputAction = iota
	inputNewSibling
	inputNewChild
	inputEditItem
)

// confirmAction tracks what the confirm dialog is for.
type confirmAction int

const (
	confirmDeleteList confirmAction = iota
	confirmDeleteItem
)

// Model is the top-level Bubble Tea model.
type Model struct {
	mode        mode
	prevMode    mode // screen behind input/confirm overlay
	width       int
	height      int
	inputAction inputAction

	// List picker state
	lists      []string
	listCursor int
	listScroll int

	// Items browser state
	list       *todo.List
	flat       []flatItem
	itemCursor int
	itemScroll int

	// Text input
	textInput textinput.Model

	// Confirm dialog
	confirmMsg       string
	confirmKind      confirmAction
	confirmTarget    string // list name for delete list
	confirmItemPath  []int  // item path for delete item
}

// New creates a new Model with default state.
func New() Model {
	ti := textinput.New()
	ti.CharLimit = 256

	s := ti.Styles()
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(colorAccent)
	s.Focused.Text = lipgloss.NewStyle().Foreground(colorPrimary)
	s.Cursor.Color = colorAccent
	ti.SetStyles(s)

	return Model{
		mode:      modeListPicker,
		textInput: ti,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadLists
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case listsLoadedMsg:
		m.lists = msg.lists
		return m, nil

	case listOpenedMsg:
		if msg.err != nil {
			return m, nil
		}
		m.list = msg.list
		m.mode = modeItems
		m.itemCursor = 0
		m.itemScroll = 0
		m.rebuildFlat()
		return m, nil

	case tea.KeyPressMsg:
		// Global quit on ctrl+c
		if msg.String() == "ctrl+c" {
			return m, m.saveAndQuit
		}
	}

	switch m.mode {
	case modeListPicker:
		return m.updateListPicker(msg)
	case modeItems:
		return m.updateItems(msg)
	case modeInput:
		return m.updateInput(msg)
	case modeConfirm:
		return m.updateConfirm(msg)
	}
	return m, nil
}

// headerLines is the number of rendered lines before the first content row
// in the panel: frame top padding (1) + title line (1) + panel border top (1).
const headerLines = 3

// visibleRows returns how many content rows fit in the panel.
func (m Model) visibleRows() int {
	return max(m.height-7, 1)
}

// halfPage returns the half-page jump distance.
func (m Model) halfPage() int {
	return max(m.visibleRows()/2, 1)
}

// clampScroll adjusts scroll so that cursor is within the visible window.
func clampScroll(cursor, scroll, visible int) int {
	if cursor < scroll {
		scroll = cursor
	}
	if cursor >= scroll+visible {
		scroll = cursor - visible + 1
	}
	return scroll
}

// panelWidth returns the inner width for the content panel.
func (m Model) panelWidth() int {
	return max(m.width-26, 30)
}

func (m Model) View() tea.View {
	if m.width == 0 {
		v := tea.NewView("Loading...")
		v.AltScreen = true
		return v
	}

	var s string
	switch m.mode {
	case modeListPicker:
		s = m.viewListPicker()
	case modeItems:
		s = m.viewItems()
	case modeInput:
		s = m.viewInput()
	case modeConfirm:
		s = m.viewConfirm()
	}

	framed := styleFrame.Render(s)
	content := lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, framed)

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// scrollInfo describes the scroll state for a panel.
type scrollInfo struct {
	showUp, showDown   bool
	thumbStart, thumbEnd int // row indices within visible window
}

// computeScroll calculates scroll indicators and scrollbar thumb position.
func computeScroll(scroll, total, visible int) scrollInfo {
	si := scrollInfo{
		showUp:   scroll > 0,
		showDown: scroll+visible < total,
	}
	thumbSize := max(visible*visible/total, 1)
	si.thumbStart = scroll * visible / total
	si.thumbEnd = si.thumbStart + thumbSize
	return si
}

// renderScrollbar adds arrow indicators and a left-side scrollbar to content lines.
// lines must have exactly `visible` entries. panelWidth is used to center arrows.
func renderScrollbar(lines []string, si scrollInfo, panelWidth int) []string {
	if !si.showUp && !si.showDown {
		return lines // no overflow, no track
	}

	if si.showUp {
		arrow := styleScrollArrow.Render("▲")
		pad := (panelWidth - 1) / 2 // rough center
		if pad < 0 {
			pad = 0
		}
		lines[0] = strings.Repeat(" ", pad) + arrow
	}
	if si.showDown && len(lines) > 0 {
		arrow := styleScrollArrow.Render("▼")
		pad := (panelWidth - 1) / 2
		if pad < 0 {
			pad = 0
		}
		lines[len(lines)-1] = strings.Repeat(" ", pad) + arrow
	}

	out := make([]string, len(lines))
	for i, line := range lines {
		var ch string
		if i >= si.thumbStart && i < si.thumbEnd {
			ch = styleScrollThumb.Render("█")
		} else {
			ch = styleScrollTrack.Render("│")
		}
		out[i] = ch + line
	}
	return out
}

// Messages

type listsLoadedMsg struct {
	lists []string
	err   error
}

type listOpenedMsg struct {
	list *todo.List
	err  error
}

func (m Model) loadLists() tea.Msg {
	names, err := todo.ListAll()
	return listsLoadedMsg{lists: names, err: err}
}

func (m Model) openList(name string) tea.Cmd {
	return func() tea.Msg {
		list, err := todo.Load(name)
		return listOpenedMsg{list: list, err: err}
	}
}

func (m Model) saveAndQuit() tea.Msg {
	if m.list != nil {
		_ = todo.Save(m.list)
	}
	return tea.QuitMsg{}
}

func (m Model) save() {
	if m.list != nil {
		_ = todo.Save(m.list)
	}
}

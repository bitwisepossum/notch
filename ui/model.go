package ui

import (
	"path/filepath"
	"strings"

	"github.com/bitwisepossum/notch/todo"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// mode is the active UI screen.
type mode int

const (
	modeListPicker mode = iota // list selection screen
	modeItems                  // item browser for an open list
	modeInput                  // text input overlay
	modeConfirm                // yes/no confirmation overlay
	modeSearch                 // search/filter input overlay
	modeSettings               // settings screen
)

// inputAction tracks what the text input is being used for.
type inputAction int

const (
	inputNewList inputAction = iota
	inputNewSibling
	inputNewChild
	inputEditItem
	inputSetDataDir
	inputRenameList
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

	// Settings
	settings       todo.Settings
	themes         []Theme
	settingsCursor int    // 0=save path, 1=theme
	defaultDataDir string // resolved OS default data dir (never changes)
	activeListDir  string // current list storage dir (custom or default)
	themesDir      string // <defaultDataDir>/themes (never changes)

	// List picker state
	lists      []string
	listCursor int
	listScroll int

	// Items browser state
	list        *todo.List
	flat        []flatItem
	itemCursor  int
	itemScroll  int
	searchQuery string // active filter; empty means no filter

	// Text input
	textInput textinput.Model

	// Confirm dialog
	confirmMsg      string
	confirmKind     confirmAction
	confirmTarget   string // list name for delete list
	confirmItemPath []int  // item path for delete item
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

// Init implements tea.Model; loads settings then the list of saved lists on startup.
func (m Model) Init() tea.Cmd {
	return m.loadSettingsCmd
}

// Update implements tea.Model; routes messages to the active mode handler.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case settingsLoadedMsg:
		m.settings = msg.settings
		m.defaultDataDir = msg.defaultDataDir
		m.activeListDir = msg.activeListDir
		m.themesDir = filepath.Join(msg.defaultDataDir, "themes")
		return m, m.loadThemesCmd

	case themesLoadedMsg:
		m.themes = msg.themes
		m.applyActiveTheme()
		return m, m.loadLists

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
		m.searchQuery = ""
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
	case modeSearch:
		return m.updateSearch(msg)
	case modeSettings:
		return m.updateSettings(msg)
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
// When total > visible, scroll arrows occupy the first and last visible rows,
// so the cursor is kept within the inner content rows.
func clampScroll(cursor, scroll, visible, total int) int {
	if cursor < scroll {
		scroll = cursor
	}
	if cursor >= scroll+visible {
		scroll = cursor - visible + 1
	}
	if total > visible {
		// ▲ arrow occupies lines[0] when scroll > 0; keep cursor below it.
		if scroll > 0 && cursor <= scroll {
			scroll = max(cursor-1, 0)
		}
		// ▼ arrow occupies lines[visible-1] when more content exists below.
		if scroll+visible < total && cursor >= scroll+visible-1 {
			scroll = cursor - visible + 2
		}
	}
	return scroll
}

// panelWidth returns the inner width for the content panel.
func (m Model) panelWidth() int {
	return max(m.width-33, 30)
}

// View implements tea.Model; renders the active screen.
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
	case modeSearch:
		s = m.viewSearch()
	case modeSettings:
		s = m.viewSettings()
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
	showUp, showDown     bool
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

	pad := (panelWidth - 1) / 2
	if pad < 0 {
		pad = 0
	}
	if si.showUp {
		lines[0] = strings.Repeat(" ", pad) + styleScrollArrow.Render("▲")
	}
	if si.showDown && len(lines) > 0 {
		lines[len(lines)-1] = strings.Repeat(" ", pad) + styleScrollArrow.Render("▼")
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

type settingsLoadedMsg struct {
	settings       todo.Settings
	defaultDataDir string
	activeListDir  string
}

type themesLoadedMsg struct {
	themes []Theme
}

type listsLoadedMsg struct {
	lists []string
	err   error
}

type listOpenedMsg struct {
	list *todo.List
	err  error
}

// loadSettingsCmd loads settings from disk and returns a settingsLoadedMsg.
func (m Model) loadSettingsCmd() tea.Msg {
	s, _ := todo.LoadSettings()
	defaultDir, _ := todo.DataDir()
	activeDir, _ := todo.ListDir()
	return settingsLoadedMsg{settings: s, defaultDataDir: defaultDir, activeListDir: activeDir}
}

// loadThemesCmd scans the themes directory and returns a themesLoadedMsg.
func (m Model) loadThemesCmd() tea.Msg {
	return themesLoadedMsg{themes: LoadThemes()}
}

// applyActiveTheme applies the theme matching settings.ActiveTheme to all style vars.
func (m *Model) applyActiveTheme() {
	for _, t := range m.themes {
		if t.Key == m.settings.ActiveTheme {
			applyTheme(t)
			return
		}
	}
	applyTheme(DefaultTheme)
}

// activeThemeIdx returns the index into m.themes of the currently active theme.
func (m Model) activeThemeIdx() int {
	for i, t := range m.themes {
		if t.Key == m.settings.ActiveTheme {
			return i
		}
	}
	return 0
}

// refreshListDir updates activeListDir from the current settings.
func (m *Model) refreshListDir() {
	if d, err := todo.ListDir(); err == nil {
		m.activeListDir = d
	}
}

// loadLists fetches all saved list names and returns a listsLoadedMsg.
func (m Model) loadLists() tea.Msg {
	names, err := todo.ListAll()
	return listsLoadedMsg{lists: names, err: err}
}

// openList returns a command that loads and parses the named list file.
func (m Model) openList(name string) tea.Cmd {
	return func() tea.Msg {
		list, err := todo.Load(name)
		return listOpenedMsg{list: list, err: err}
	}
}

// saveAndQuit persists the open list (if any) and signals the program to exit.
func (m Model) saveAndQuit() tea.Msg {
	if m.list != nil {
		_ = todo.Save(m.list)
	}
	return tea.QuitMsg{}
}

// save persists the open list to disk, silently ignoring errors.
func (m Model) save() {
	if m.list != nil {
		_ = todo.Save(m.list)
	}
}

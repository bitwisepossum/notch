package ui

import (
	"log/slog"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bitwisepossum/notch/todo"
)

// listEntry holds the name and progress summary for one list.
type listEntry struct {
	name       string
	totalItems int
	doneItems  int
}

// mode is the active UI screen.
type mode int

const (
	modeListPicker mode = iota // list selection screen
	modeItems                  // item browser for an open list
	modeInput                  // text input overlay
	modeConfirm                // yes/no confirmation overlay
	modeSearch                 // search/filter input overlay
	modeSettings               // settings screen
	modeLogViewer              // log file viewer
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
	inputSetDeadline
	inputQuickAdd
)

// confirmAction tracks what the confirm dialog is for.
type confirmAction int

const (
	confirmDeleteList confirmAction = iota
	confirmDeleteItem
	confirmClearLog
)

// listPicker holds list selection screen state.
type listPicker struct {
	lists  []listEntry
	cursor int
	scroll int
}

// itemBrowser holds item browsing/editing state for an open list.
type itemBrowser struct {
	list          *todo.List
	flat          []flatItem
	cursor        int
	scroll        int
	searchQuery   string          // active filter; empty means no filter
	preSearchItem *todo.Item      // item under cursor when search was opened
	folded        map[string]bool // paths of collapsed items
	hideDone      bool            // when true, completed items are filtered out
	undoStack     []snapshot
	redoStack     []snapshot
}

// inputState holds text input overlay state.
type inputState struct {
	textInput textinput.Model
	action    inputAction
	err       string // non-empty while input overlay shows an error
}

// confirmState holds confirmation dialog state.
type confirmState struct {
	msg      string
	kind     confirmAction
	target   string // list name for delete list
	itemPath []int  // item path for delete item
}

// logViewer holds log file viewer state.
type logViewer struct {
	lines  []string // cached log lines
	cursor int      // scroll position (top visible line)
}

// Model is the top-level Bubble Tea model.
type Model struct {
	mode     mode
	prevMode mode // screen behind input/confirm overlay
	width    int
	height   int

	// Settings & UI state
	settings       todo.Settings
	uiState        UIState
	themes         []Theme
	settingsCursor int    // 0=save path, 1=theme
	defaultDataDir string // resolved OS default data dir (never changes)
	activeListDir  string // current list storage dir (custom or default)
	themesDir      string // <defaultDataDir>/themes (never changes)
	flashErr       string // non-empty: IO error shown in status bar
	showHelp       bool   // whether the help sidebar is visible

	lp      listPicker
	ib      itemBrowser
	input   inputState
	confirm confirmState
	log     logViewer
}

// New creates a new Model with default state.
func New() Model {
	ti := textinput.New()
	ti.CharLimit = 256

	s := ti.Styles()
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(colorAccent)
	s.Focused.Text = lipgloss.NewStyle().Foreground(colorPrimary)
	s.Cursor.Color = colorAccent
	// Blinking cursors cause periodic redraws. When the underlying view is
	// relatively heavy (e.g. long item lists with rich rendering), those redraws
	// can make the UI feel "hung" as soon as any input is focused.
	s.Cursor.Blink = false
	ti.SetStyles(s)

	return Model{
		mode:  modeListPicker,
		input: inputState{textInput: ti},
		ib:    itemBrowser{folded: make(map[string]bool)},
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
		switch m.mode {
		case modeItems:
			if len(m.ib.flat) > 0 {
				m.ib.cursor = min(m.ib.cursor, len(m.ib.flat)-1)
			}
			m.ib.scroll = clampScroll(m.ib.cursor, m.ib.scroll, m.visibleRows(), len(m.ib.flat))
		case modeListPicker:
			if len(m.lp.lists) > 0 {
				m.lp.cursor = min(m.lp.cursor, len(m.lp.lists)-1)
			}
			m.lp.scroll = clampScroll(m.lp.cursor, m.lp.scroll, m.visibleRows(), len(m.lp.lists))
		case modeSettings:
			m.settingsCursor = min(m.settingsCursor, settingsRowCount-1)
		case modeLogViewer:
			m.log.cursor = min(m.log.cursor, max(len(m.log.lines)-1, 0))
		}
		return m, nil

	case settingsLoadedMsg:
		m.settings = msg.settings
		m.uiState = msg.uiState
		m.defaultDataDir = msg.defaultDataDir
		m.activeListDir = msg.activeListDir
		m.themesDir = filepath.Join(msg.defaultDataDir, "themes")
		if m.uiState.ShowHelp == nil {
			m.showHelp = true // first start: show help by default
		} else {
			m.showHelp = *m.uiState.ShowHelp
		}
		m.setFlash(todo.InitLogger(m.settings.LogLevel))
		return m, m.loadThemesCmd

	case themesLoadedMsg:
		m.themes = msg.themes
		m.applyActiveTheme()
		return m, m.loadLists

	case listsLoadedMsg:
		m.lp.lists = msg.lists
		// Remove fold state for lists that no longer exist on disk.
		if len(m.uiState.FoldState) > 0 {
			existing := make(map[string]bool, len(m.lp.lists))
			for _, e := range m.lp.lists {
				existing[e.name] = true
			}
			us := m.uiState
			changed := false
			for name := range us.FoldState {
				if !existing[name] {
					delete(us.FoldState, name)
					changed = true
				}
			}
			if changed {
				_ = saveUIState(us)
				m.uiState = us
			}
		}
		return m, nil

	case listOpenedMsg:
		if msg.err != nil {
			return m, nil
		}
		m.ib.list = msg.list
		m.mode = modeItems
		m.ib.cursor = 0
		m.ib.scroll = 0
		m.ib.searchQuery = ""
		m.ib.hideDone = false
		m.ib.folded = make(map[string]bool)
		m.ib.undoStack = m.ib.undoStack[:0]
		m.ib.redoStack = m.ib.redoStack[:0]
		m.loadFoldState()
		m.rebuildFlat()
		return m, nil

	case tea.KeyPressMsg:
		m.flashErr = "" // dismiss any IO error on next input
		// Global quit on ctrl+c
		if msg.String() == "ctrl+c" {
			return m, m.saveAndQuit
		}
		// Global F1: toggle help sidebar
		if msg.String() == "f1" {
			m.showHelp = !m.showHelp
			return m, nil
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
	case modeLogViewer:
		return m.updateLogViewer(msg)
	}
	return m, nil
}

// headerLines is the number of rendered lines before the first content row
// in the panel: frame top padding (1) + title line (1) + panel border top (1).
const headerLines = 3

// visibleRows returns how many content rows fit in the panel.
// Layout: 1 frame top + 1 title + 1 panel top border + content + 1 panel bottom border + 1 status bar + 1 frame bottom = height - 8 for items.
// List picker and settings share the same count (no status bar, but close enough in practice).
func (m Model) visibleRows() int {
	return max(m.height-8, 1)
}

// helpHeight returns the max number of content lines the help sidebar can render
// without overflowing the terminal. Accounts for: frame PaddingTop(1), title
// line (1), help panel PaddingTop(1), and one row of bottom margin (1).
func (m Model) helpHeight() int {
	return max(m.height-4, 1)
}

// halfPage returns the half-page jump distance.
func (m Model) halfPage() int {
	return max(m.visibleRows()/2, 1)
}

// clampScroll adjusts scroll so that cursor is within the visible window,
// then accounts for scroll arrow indicators.
func clampScroll(cursor, scroll, visible, total int) int {
	scroll = basicClamp(cursor, scroll, visible)
	scroll = adjustForArrows(cursor, scroll, visible, total)
	return scroll
}

// basicClamp ensures cursor is within [scroll, scroll+visible).
func basicClamp(cursor, scroll, visible int) int {
	if cursor < scroll {
		scroll = cursor
	}
	if cursor >= scroll+visible {
		scroll = cursor - visible + 1
	}
	return scroll
}

// adjustForArrows shifts scroll when scroll arrows occupy the first/last
// visible rows, keeping the cursor off those indicator rows.
func adjustForArrows(cursor, scroll, visible, total int) int {
	if total <= visible {
		return scroll
	}
	// ▲ arrow occupies lines[0] when scroll > 0; keep cursor below it.
	if scroll > 0 && cursor <= scroll {
		scroll = max(cursor-1, 0)
	}
	// ▼ arrow occupies lines[visible-1] when more content exists below.
	if scroll+visible < total && cursor >= scroll+visible-1 {
		scroll = cursor - visible + 2
	}
	return scroll
}

// activeHelpGroups returns the helpGroups that will be rendered for the current mode.
func (m Model) activeHelpGroups() []helpGroup {
	switch m.mode {
	case modeItems, modeSearch:
		return itemsHelp
	case modeSettings:
		return settingsHelp
	case modeLogViewer:
		return logViewerHelp
	default:
		return listHelp
	}
}

// helpTwoCol reports whether the help sidebar will need two columns given the
// current terminal height. Compact rendering (no blanks, no headers) reduces
// the line count to the bare pair count; two columns are needed if that still
// exceeds helpHeight.
func (m Model) helpTwoCol() bool {
	if !m.showHelp {
		return false
	}
	n := 0
	for _, g := range m.activeHelpGroups() {
		n += len(g.pairs)
	}
	return n > m.helpHeight()
}

// panelWidth returns the inner width for the content panel.
// When the help sidebar is hidden the panel expands to fill most of the terminal.
func (m Model) panelWidth() int {
	if m.showHelp {
		if m.helpTwoCol() {
			// Two-column help is ~2× the single-column width; the offset doubles
			// from 33 to 63 (2 PaddingLeft + col2_content + 2 sep + col1_content).
			return max(m.width-63, 30)
		}
		return max(m.width-33, 30)
	}
	// Subtract 5 (not 4) to account for the frame's PaddingLeft(1), so the
	// panel + frame together fit exactly within m.width columns.
	return max(m.width-5, 30)
}

// helpHint returns a styled "F1 help" indicator, or "" when help is already shown.
// Used to show a discoverable hint in the bottom line of each screen.
func (m Model) helpHint() string {
	if m.showHelp {
		return ""
	}
	return styleHelpKey.Render("F1") + styleHelpDesc.Render(" help")
}

// right-align suffix within panelWidth
func (m Model) rightAlign(s, suffix string) string {
	if suffix == "" {
		return s
	}
	totalWidth := m.panelWidth()
	gap := totalWidth - lipgloss.Width(s) - lipgloss.Width(suffix)
	if gap > 0 {
		return s + strings.Repeat(" ", gap) + suffix
	}
	return s
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
		// viewConfirm renders the background and overlays the popup itself.
		// Return early — no frame wrapping needed.
		v := tea.NewView(m.viewConfirm())
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	case modeSearch:
		s = m.viewSearch()
	case modeSettings:
		s = m.viewSettings()
	case modeLogViewer:
		s = m.viewLogViewer()
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
	if total == 0 {
		return scrollInfo{}
	}
	si := scrollInfo{
		showUp:   scroll > 0,
		showDown: scroll+visible < total,
	}
	thumbSize := max(visible*visible/total, 1)
	si.thumbStart = scroll * visible / total
	si.thumbEnd = si.thumbStart + thumbSize
	return si
}

// renderProgress renders a fixed-width progress bar using the active theme characters.
// filled portion is styled with colorAccent, empty with colorSeparator.
func renderProgress(done, total, width int) string {
	if total <= 0 || width <= 0 {
		return lipgloss.NewStyle().Foreground(colorSeparator).Render(strings.Repeat(charBarEmpty, width))
	}
	filled := min(done*width/total, width)
	bar := lipgloss.NewStyle().Foreground(colorAccent).Render(strings.Repeat(charBarFilled, filled)) +
		lipgloss.NewStyle().Foreground(colorSeparator).Render(strings.Repeat(charBarEmpty, width-filled))
	return bar
}

// renderScrollbar adds arrow indicators and a left-side scrollbar to content lines.
// lines must have exactly `visible` entries. panelWidth is used to center arrows.
func renderScrollbar(lines []string, si scrollInfo, panelWidth int) []string {
	if !si.showUp && !si.showDown {
		return lines // no overflow, no track
	}

	pad := max((panelWidth-1)/2, 0)
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
			ch = styleScrollTrack.Render("▕")
		}
		out[i] = ch + line
	}
	return out
}

// Messages

type settingsLoadedMsg struct {
	settings       todo.Settings
	uiState        UIState
	defaultDataDir string
	activeListDir  string
}

type themesLoadedMsg struct {
	themes []Theme
}

type listsLoadedMsg struct {
	lists []listEntry
}

type listOpenedMsg struct {
	list *todo.List
	err  error
}

// loadSettingsCmd loads settings from disk and returns a settingsLoadedMsg.
func (m Model) loadSettingsCmd() tea.Msg {
	s, _ := todo.LoadSettings()
	us, _ := loadUIState()
	defaultDir, _ := todo.DataDir()
	activeDir, _ := todo.ListDir()
	return settingsLoadedMsg{settings: s, uiState: us, defaultDataDir: defaultDir, activeListDir: activeDir}
}

// loadThemesCmd scans the themes directory and returns a themesLoadedMsg.
func (m Model) loadThemesCmd() tea.Msg {
	return themesLoadedMsg{themes: LoadThemes()}
}

// applyActiveTheme applies the theme matching settings.ActiveTheme to all style vars.
func (m *Model) applyActiveTheme() {
	for _, t := range m.themes {
		if t.Key == m.settings.ActiveTheme {
			if t.Error != "" {
				todo.LogError("active theme invalid, using default",
					slog.String("theme", t.Key),
					slog.String("err", t.Error))
				applyTheme(DefaultTheme)
			} else {
				applyTheme(t)
			}
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

// loadListEntries fetches all list names and loads each one to compute its
// item counts. Returns nil on ListAll failure.
func loadListEntries() []listEntry {
	names, err := todo.ListAll()
	if err != nil {
		return nil
	}
	entries := make([]listEntry, len(names))
	for i, name := range names {
		entries[i].name = name
		if list, err := todo.Load(name); err == nil {
			entries[i].totalItems, entries[i].doneItems = subtreeCount(&todo.Item{Children: list.Items})
		}
	}
	return entries
}

// hasListNamed reports whether any entry has the given name.
func hasListNamed(lists []listEntry, name string) bool {
	for _, e := range lists {
		if e.name == name {
			return true
		}
	}
	return false
}

// loadLists fetches all saved lists and returns a listsLoadedMsg.
func (m Model) loadLists() tea.Msg {
	return listsLoadedMsg{lists: loadListEntries()}
}

// openList returns a command that loads and parses the named list file.
func (m Model) openList(name string) tea.Cmd {
	return func() tea.Msg {
		list, err := todo.Load(name)
		return listOpenedMsg{list: list, err: err}
	}
}

// persistShowHelp writes the current showHelp preference to UI state.
func (m Model) persistShowHelp() {
	val := m.showHelp
	us := m.uiState
	us.ShowHelp = &val
	_ = saveUIState(us)
}

// saveAndQuit persists the open list (if any) and signals the program to exit.
func (m Model) saveAndQuit() tea.Msg {
	m.persistShowHelp()
	if m.ib.list != nil {
		_ = todo.Save(m.ib.list)
		m.saveFoldState()
	}
	todo.CloseLogger()
	return tea.QuitMsg{}
}

// save persists the open list to disk and returns any write error.
func (m *Model) save() error {
	if m.ib.list != nil {
		return todo.Save(m.ib.list)
	}
	return nil
}

// setFlash records an IO error for display in the status bar.
// Passing nil is a no-op.
func (m *Model) setFlash(err error) {
	if err != nil {
		m.flashErr = "Error: " + err.Error()
		todo.LogError("io error", slog.String("err", err.Error()))
	}
}

// saveFlash saves the open list and records any write error as a flash message.
func (m *Model) saveFlash() { m.setFlash(m.save()) }

// saveFoldState writes the current fold state for the open list to ui_state.json.
// It stores index-based path keys alongside a content hash of the list so that
// stale state can be detected on next load.
func (m Model) saveFoldState() {
	if m.ib.list == nil {
		return
	}
	var paths []string
	for k := range m.ib.folded {
		paths = append(paths, k)
	}
	us := m.uiState
	if len(paths) == 0 {
		if us.FoldState != nil {
			delete(us.FoldState, m.ib.list.Name)
		}
	} else {
		if us.FoldState == nil {
			us.FoldState = make(map[string]SavedFolds)
		}
		us.FoldState[m.ib.list.Name] = SavedFolds{
			Hash:  m.ib.list.Hash(),
			Paths: paths,
		}
	}
	_ = saveUIState(us)
}

// loadFoldState restores fold state for the open list from UI state.
// If the stored hash doesn't match the current list content, fold state is discarded.
func (m *Model) loadFoldState() {
	if m.ib.list == nil {
		return
	}
	entry, ok := m.uiState.FoldState[m.ib.list.Name]
	if !ok || len(entry.Paths) == 0 {
		return
	}
	if entry.Hash != m.ib.list.Hash() {
		return
	}
	for _, p := range entry.Paths {
		m.ib.folded[p] = true
	}
}

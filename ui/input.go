package ui

import (
	"log/slog"
	"strings"
	"time"

	"github.com/bitwisepossum/notch/todo"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// updateInput handles messages while the text input overlay is active.
func (m Model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(m.input.textInput.Value())
			m.input.err = ""
			// inputSetDeadline allows empty val (clears the deadline).
			if val != "" || m.input.action == inputSetDeadline {
				m.commitInput(val)
			}
			if m.input.err == "" {
				m.input.textInput.Blur()
				m.mode = m.prevMode
			}
			return m, nil
		case "esc":
			m.input.err = ""
			m.input.textInput.Blur()
			m.mode = m.prevMode
			return m, nil
		default:
			// User typed something — clear any standing error.
			m.input.err = ""
		}
	}

	var cmd tea.Cmd
	m.input.textInput, cmd = m.input.textInput.Update(msg)
	return m, cmd
}

// commitInput applies the submitted text value based on the active inputAction.
func (m *Model) commitInput(val string) {
	switch m.input.action {
	case inputNewList:
		m.commitNewList(val)
	case inputNewSibling:
		m.commitNewSibling(val)
	case inputNewChild:
		m.commitNewChild(val)
	case inputEditItem:
		m.commitEditItem(val)
	case inputSetDataDir:
		m.commitSetDataDir(val)
	case inputRenameList:
		m.commitRenameList(val)
	case inputQuickAdd:
		m.commitQuickAdd(val)
	case inputSetDeadline:
		m.commitSetDeadline(val)
	}
}

func (m *Model) commitNewList(val string) {
	if hasListNamed(m.lp.lists, val) {
		m.input.err = "a list named " + val + " already exists"
		return
	}
	newList := &todo.List{Name: val}
	m.setFlash(todo.Save(newList))
	todo.LogEvent("list created", slog.String("name", val))
	m.lp.lists = loadListEntries()
	for i, e := range m.lp.lists {
		if e.name == val {
			m.lp.cursor = i
			break
		}
	}
}

func (m *Model) commitNewSibling(val string) {
	todo.LogEvent("item added", slog.String("list", m.ib.list.Name), slog.String("name", val))
	m.pushUndo()
	if len(m.ib.flat) == 0 {
		m.ib.list.Add(nil, val)
	} else {
		fi := m.ib.flat[m.ib.cursor]
		parentPath := fi.path[:len(fi.path)-1]
		siblingIdx := fi.path[len(fi.path)-1] + 1
		m.ib.list.Add(parentPath, val)

		parent := parentPath
		items := m.ib.list.Items
		if len(parent) > 0 {
			p := resolveItem(items, parent)
			if p != nil {
				items = p.Children
			}
		}
		lastIdx := len(items) - 1
		if lastIdx != siblingIdx {
			from := make([]int, len(parent)+1)
			copy(from, parent)
			from[len(parent)] = lastIdx
			to := make([]int, len(parent)+1)
			copy(to, parent)
			to[len(parent)] = siblingIdx
			_ = m.ib.list.Move(from, to)
		}
	}
	m.saveFlash()
	m.rebuildFlat()
	m.ib.cursor = min(m.ib.cursor+1, len(m.ib.flat)-1)
}

func (m *Model) commitNewChild(val string) {
	if len(m.ib.flat) == 0 {
		return
	}
	todo.LogEvent("item added", slog.String("list", m.ib.list.Name), slog.String("name", val))
	m.pushUndo()
	fi := m.ib.flat[m.ib.cursor]
	m.ib.list.Add(fi.path, val)
	newChild := fi.item.Children[len(fi.item.Children)-1]
	m.saveFlash()
	m.rebuildFlat()
	m.followItem(newChild)
}

func (m *Model) commitEditItem(val string) {
	if len(m.ib.flat) == 0 {
		return
	}
	todo.LogEvent("item edited", slog.String("list", m.ib.list.Name), slog.String("name", val))
	m.pushUndo()
	fi := m.ib.flat[m.ib.cursor]
	_ = m.ib.list.Edit(fi.path, val)
	m.saveFlash()
	m.rebuildFlat()
}

func (m *Model) commitSetDataDir(val string) {
	todo.LogEvent("data dir changed", slog.String("path", todo.SanitizePath(val)))
	m.settings.CustomDataDir = val
	m.setFlash(todo.SaveSettings(m.settings))
	todo.InvalidateListDir()
	m.refreshListDir()
	m.lp.lists = loadListEntries()
	m.lp.cursor = 0
	m.lp.scroll = 0
}

func (m *Model) commitRenameList(val string) {
	oldName := m.lp.lists[m.lp.cursor].name
	if oldName != val && hasListNamed(m.lp.lists, val) {
		m.input.err = "a list named " + val + " already exists"
		return
	}
	if oldName == val {
		return
	}
	list, err := todo.Load(oldName)
	if err != nil {
		todo.LogError("load list for rename", slog.String("list", oldName), slog.String("err", err.Error()))
		return
	}
	list.Rename(val)
	if err := todo.Save(list); err != nil {
		todo.LogError("save renamed list", slog.String("list", val), slog.String("err", err.Error()))
		return
	}
	todo.LogEvent("list renamed", slog.String("from", oldName), slog.String("to", val))
	m.setFlash(todo.Delete(oldName))
	us := m.uiState
	if us.FoldState != nil {
		if entry, ok := us.FoldState[oldName]; ok {
			delete(us.FoldState, oldName)
			us.FoldState[val] = entry
			_ = saveUIState(us)
			m.uiState = us
		}
	}
	m.lp.lists = loadListEntries()
	for i, e := range m.lp.lists {
		if e.name == val {
			m.lp.cursor = i
			break
		}
	}
}

func (m *Model) commitQuickAdd(val string) {
	name := m.lp.lists[m.lp.cursor].name
	list, err := todo.Load(name)
	if err != nil {
		todo.LogError("load list for quick add", slog.String("list", name), slog.String("err", err.Error()))
		return
	}
	list.Add(nil, val)
	m.setFlash(todo.Save(list))
}

func (m *Model) commitSetDeadline(val string) {
	if len(m.ib.flat) == 0 {
		return
	}
	fi := m.ib.flat[m.ib.cursor]
	m.pushUndo()
	if val == "" {
		_ = m.ib.list.SetDeadline(fi.path, time.Time{})
		todo.LogEvent("deadline cleared", slog.String("list", m.ib.list.Name), slog.String("item", fi.item.Text))
	} else {
		d, err := time.Parse(deadlineLayout(m.settings), val)
		if err != nil {
			m.input.err = "use " + deadlineLabel(m.settings) + " (e.g. " + deadlineHint(m.settings) + ")"
			return
		}
		_ = m.ib.list.SetDeadline(fi.path, d)
		todo.LogEvent("deadline set", slog.String("list", m.ib.list.Name), slog.String("item", fi.item.Text), slog.String("deadline", val))
	}
	m.saveFlash()
	m.rebuildFlat()
}

// viewInput renders the underlying screen with the text input prompt below it.
func (m Model) viewInput() string {
	var b strings.Builder

	// Render the underlying screen.
	var underlying string
	switch m.prevMode {
	case modeListPicker:
		underlying = m.viewListPicker()
	case modeItems:
		underlying = m.viewItems()
	case modeSettings:
		underlying = m.viewSettings()
	}

	// The underlying views generally try to fill the entire terminal height.
	// When we add an input row + help row, the bottom of the screen can get
	// clipped, making the input appear "missing". Trim the underlying render to
	// leave room for the input UI.
	reserved := 3 // blank line + input line + help line
	maxUnderlying := max(m.height-reserved, 1)
	lines := strings.Split(underlying, "\n")
	if len(lines) > maxUnderlying {
		lines = lines[:maxUnderlying]
	}
	b.WriteString(strings.Join(lines, "\n"))
	b.WriteString("\n")

	var prompt string
	switch m.input.action {
	case inputNewList:
		prompt = "New list: "
	case inputNewSibling:
		prompt = "New item: "
	case inputNewChild:
		prompt = "New child: "
	case inputEditItem:
		prompt = "Edit: "
	case inputSetDataDir:
		prompt = "Save path: "
	case inputRenameList:
		prompt = "Rename: "
	case inputSetDeadline:
		prompt = "Deadline (" + deadlineLabel(m.settings) + ", empty=clear): "
	case inputQuickAdd:
		prompt = "Add to " + m.lp.lists[m.lp.cursor].name + ": "
	}
	errStr := ""
	if m.input.err != "" {
		errStr = "   " + styleConfirm.Render(m.input.err)
	}
	b.WriteString(stylePrompt.Render(prompt) + m.input.textInput.View() + errStr + "\n")
	b.WriteString(renderHelp(inputHelp, 0))
	return b.String()
}

// updateConfirm handles y/n key presses in the confirmation dialog.
func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "y":
			switch m.confirm.kind {
			case confirmClearLog:
				m.setFlash(todo.ClearLog())
				m.log.lines = nil
				m.log.cursor = 0
				m.mode = m.prevMode
				return m, nil
			case confirmDeleteList:
				todo.LogEvent("list deleted", slog.String("name", m.confirm.target))
				m.setFlash(todo.Delete(m.confirm.target))
				// Remove any persisted fold state for the deleted list.
				us := m.uiState
				if us.FoldState != nil {
					if _, ok := us.FoldState[m.confirm.target]; ok {
						delete(us.FoldState, m.confirm.target)
						_ = saveUIState(us) // fold state only; non-critical
						m.uiState = us
					}
				}
				m.lp.lists = loadListEntries()
				if m.lp.cursor >= len(m.lp.lists) && m.lp.cursor > 0 {
					m.lp.cursor = len(m.lp.lists) - 1
				}
			case confirmDeleteItem:
				if len(m.ib.flat) > 0 {
					fi := m.ib.flat[m.ib.cursor]
					todo.LogEvent("item deleted", slog.String("list", m.ib.list.Name), slog.String("name", fi.item.Text))
				}
				m.pushUndo()
				saved := m.snapshotFoldedItems()
				_ = m.ib.list.Remove(m.confirm.itemPath)
				m.rebuildFoldedFromPointers(saved)
				m.saveFlash()
				m.rebuildFlat()
				if m.ib.cursor >= len(m.ib.flat) && m.ib.cursor > 0 {
					m.ib.cursor = len(m.ib.flat) - 1
				}
			}
			m.mode = m.prevMode
			return m, nil
		case "n", "esc":
			m.mode = m.prevMode
			return m, nil
		}
	}
	return m, nil
}

// viewConfirm renders a centered popup dialog overlaid on the underlying screen.
func (m Model) viewConfirm() string {
	// Render background at terminal dimensions.
	var underlying string
	switch m.prevMode {
	case modeListPicker:
		underlying = m.viewListPicker()
	case modeItems:
		underlying = m.viewItems()
	case modeLogViewer:
		underlying = m.viewLogViewer()
	}
	bg := lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top,
		styleFrame.Render(underlying))

	// Build popup.
	msg := styleConfirm.Render(m.confirm.msg)
	hint := styleHelpKey.Render("y") + styleHelpDesc.Render("  /  ") + styleHelpKey.Render("n")
	innerWidth := max(lipgloss.Width(msg), lipgloss.Width(hint))
	popup := stylePanel.Width(innerWidth).Render(msg + "\n\n" + hint)

	// Compose layers: background + popup centered within the panel area.
	popupW := lipgloss.Width(popup)
	popupH := lipgloss.Height(popup)
	// The panel occupies frameLeft(2) + border(1) + panelWidth + border(1).
	// Center the popup within that region.
	panelArea := 2 + 2 + m.panelWidth()
	bgLayer := lipgloss.NewLayer(bg)
	fgLayer := lipgloss.NewLayer(popup).
		X((panelArea - popupW) / 2).
		Y((m.height - popupH) / 2).
		Z(1)

	return lipgloss.NewCompositor(bgLayer, fgLayer).Render()
}

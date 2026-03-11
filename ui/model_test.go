package ui

import (
	"testing"

	"github.com/bitwisepossum/notch/todo"
	tea "charm.land/bubbletea/v2"
)

// key constructs a tea.KeyPressMsg for a printable rune.
func key(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)})
}

// specialKey constructs a tea.KeyPressMsg for a special key (enter, esc, etc).
func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code})
}

// ctrlKey constructs a tea.KeyPressMsg with the ctrl modifier.
func ctrlKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Mod: tea.ModCtrl})
}

// newTestModel creates a Model with a loaded list, ready for item browser tests.
// Bypasses filesystem — the list is constructed in-memory.
func newTestModel() Model {
	m := New()
	m.width = 80
	m.height = 24
	m.mode = modeItems
	m.list = &todo.List{
		Name: "test",
		Items: []*todo.Item{
			{Text: "First", Children: []*todo.Item{
				{Text: "Child A"},
				{Text: "Child B", Done: true},
			}},
			{Text: "Second"},
			{Text: "Third", Done: true},
		},
	}
	m.rebuildFlat()
	return m
}

// newListPickerModel creates a Model in list picker mode with some list names.
func newListPickerModel() Model {
	m := New()
	m.width = 80
	m.height = 24
	m.mode = modeListPicker
	m.lists = []listEntry{{name: "Alpha"}, {name: "Beta"}, {name: "Gamma"}, {name: "Delta"}}
	return m
}

// update is a shorthand that calls m.Update and asserts back to Model.
func update(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	result, _ := m.Update(msg)
	return result.(Model)
}

func TestItemsCursorNavigation(t *testing.T) {
	tests := []struct {
		name   string
		start  int
		msg    tea.Msg
		expect int
	}{
		{"j moves down", 0, key('j'), 1},
		{"k moves up", 2, key('k'), 1},
		{"down arrow", 0, specialKey(tea.KeyDown), 1},
		{"up arrow", 2, specialKey(tea.KeyUp), 1},
		{"j at bottom stays", 4, key('j'), 4},
		{"k at top stays", 0, key('k'), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel()
			m.itemCursor = tt.start
			m = update(t, m, tt.msg)
			if m.itemCursor != tt.expect {
				t.Errorf("cursor = %d, want %d", m.itemCursor, tt.expect)
			}
		})
	}
}

func TestItemsToggleDone(t *testing.T) {
	m := newTestModel()
	m.itemCursor = 0 // "First" — undone

	m = update(t, m, specialKey(tea.KeySpace))

	if !m.list.Items[0].Done {
		t.Error("expected First to be done after space")
	}

	// Toggle back.
	m = update(t, m, specialKey(tea.KeyEnter))
	if m.list.Items[0].Done {
		t.Error("expected First to be undone after enter")
	}
}

func TestItemsFoldUnfold(t *testing.T) {
	m := newTestModel()
	m.itemCursor = 0 // "First" has children, expanded by default.

	initialLen := len(m.flat)

	// Fold with 'f'.
	m = update(t, m, key('f'))
	if !m.folded[pathKey([]int{0})] {
		t.Error("expected First to be folded")
	}
	if len(m.flat) >= initialLen {
		t.Error("expected fewer flat items after fold")
	}

	// Unfold with 'f'.
	m = update(t, m, key('f'))
	if m.folded[pathKey([]int{0})] {
		t.Error("expected First to be unfolded")
	}
	if len(m.flat) != initialLen {
		t.Errorf("expected %d flat items, got %d", initialLen, len(m.flat))
	}
}

func TestItemsFoldLeft(t *testing.T) {
	m := newTestModel()
	m.itemCursor = 0 // "First", expanded

	// Left arrow folds.
	m = update(t, m, specialKey(tea.KeyLeft))
	if !m.folded[pathKey([]int{0})] {
		t.Error("expected First to be folded via left arrow")
	}

	// Left again on folded item with depth 0 — stays.
	m = update(t, m, specialKey(tea.KeyLeft))
	if m.itemCursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m.itemCursor)
	}
}

func TestItemsFoldRight(t *testing.T) {
	m := newTestModel()
	m.itemCursor = 0
	m.folded[pathKey([]int{0})] = true
	m.rebuildFlat()

	// Right arrow unfolds.
	m = update(t, m, specialKey(tea.KeyRight))
	if m.folded[pathKey([]int{0})] {
		t.Error("expected First to be unfolded via right arrow")
	}
}

func TestItemsFoldAll(t *testing.T) {
	m := newTestModel()

	// Z folds all.
	m = update(t, m, key('Z'))
	if len(m.folded) == 0 {
		t.Error("expected some items folded after Z")
	}

	// Z again unfolds all.
	m = update(t, m, key('Z'))
	if len(m.folded) != 0 {
		t.Error("expected all items unfolded after second Z")
	}
}

func TestItemsHideDone(t *testing.T) {
	m := newTestModel()
	before := len(m.flat)

	// H toggles hide done.
	m = update(t, m, key('H'))
	if !m.hideDone {
		t.Error("expected hideDone=true after H")
	}
	if len(m.flat) >= before {
		t.Error("expected fewer items with done hidden")
	}

	// H again shows them.
	m = update(t, m, key('H'))
	if m.hideDone {
		t.Error("expected hideDone=false after second H")
	}
	if len(m.flat) != before {
		t.Errorf("expected %d items, got %d", before, len(m.flat))
	}
}

func TestItemsUndo(t *testing.T) {
	m := newTestModel()
	m.itemCursor = 0
	original := m.list.Items[0].Done

	// Toggle done, then undo.
	m = update(t, m, specialKey(tea.KeySpace))
	if m.list.Items[0].Done == original {
		t.Fatal("toggle should have changed done state")
	}

	m = update(t, m, key('u'))
	if m.list.Items[0].Done != original {
		t.Error("undo should restore original done state")
	}
}

func TestItemsRedo(t *testing.T) {
	m := newTestModel()
	m.itemCursor = 0

	m = update(t, m, specialKey(tea.KeySpace)) // toggle
	toggled := m.list.Items[0].Done
	m = update(t, m, key('u'))                  // undo
	m = update(t, m, ctrlKey('r'))              // redo
	if m.list.Items[0].Done != toggled {
		t.Error("redo should restore toggled state")
	}
}

func TestItemsExitToListPicker(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.Msg
	}{
		{"q exits", key('q')},
		{"esc exits", specialKey(tea.KeyEscape)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel()
			m = update(t, m, tt.msg)
			if m.mode != modeListPicker {
				t.Errorf("mode = %d, want modeListPicker", m.mode)
			}
			if m.list != nil {
				t.Error("list should be nil after exit")
			}
		})
	}
}

func TestItemsSearchMode(t *testing.T) {
	m := newTestModel()
	m = update(t, m, key('/'))
	if m.mode != modeSearch {
		t.Errorf("mode = %d, want modeSearch", m.mode)
	}
}

func TestItemsAddSibling(t *testing.T) {
	m := newTestModel()
	m = update(t, m, key('a'))
	if m.mode != modeInput {
		t.Errorf("mode = %d, want modeInput", m.mode)
	}
	if m.inputAction != inputNewSibling {
		t.Errorf("inputAction = %d, want inputNewSibling", m.inputAction)
	}
}

func TestItemsAddChild(t *testing.T) {
	m := newTestModel()
	m = update(t, m, key('A'))
	if m.mode != modeInput {
		t.Errorf("mode = %d, want modeInput", m.mode)
	}
	if m.inputAction != inputNewChild {
		t.Errorf("inputAction = %d, want inputNewChild", m.inputAction)
	}
}

func TestItemsEdit(t *testing.T) {
	m := newTestModel()
	m = update(t, m, key('e'))
	if m.mode != modeInput {
		t.Errorf("mode = %d, want modeInput", m.mode)
	}
	if m.inputAction != inputEditItem {
		t.Errorf("inputAction = %d, want inputEditItem", m.inputAction)
	}
}

func TestItemsDelete(t *testing.T) {
	m := newTestModel()
	m = update(t, m, key('d'))
	if m.mode != modeConfirm {
		t.Errorf("mode = %d, want modeConfirm", m.mode)
	}
	if m.confirmKind != confirmDeleteItem {
		t.Errorf("confirmKind = %d, want confirmDeleteItem", m.confirmKind)
	}
}

func TestItemsDeadline(t *testing.T) {
	m := newTestModel()
	m = update(t, m, key('t'))
	if m.mode != modeInput {
		t.Errorf("mode = %d, want modeInput", m.mode)
	}
	if m.inputAction != inputSetDeadline {
		t.Errorf("inputAction = %d, want inputSetDeadline", m.inputAction)
	}
}

func TestListPickerNavigation(t *testing.T) {
	tests := []struct {
		name   string
		start  int
		msg    tea.Msg
		expect int
	}{
		{"j moves down", 0, key('j'), 1},
		{"k moves up", 2, key('k'), 1},
		{"j at bottom stays", 3, key('j'), 3},
		{"k at top stays", 0, key('k'), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newListPickerModel()
			m.listCursor = tt.start
			m = update(t, m, tt.msg)
			if m.listCursor != tt.expect {
				t.Errorf("cursor = %d, want %d", m.listCursor, tt.expect)
			}
		})
	}
}

func TestListPickerSettings(t *testing.T) {
	m := newListPickerModel()
	m = update(t, m, key('s'))
	if m.mode != modeSettings {
		t.Errorf("mode = %d, want modeSettings", m.mode)
	}
}

func TestListPickerNewList(t *testing.T) {
	m := newListPickerModel()
	m = update(t, m, key('n'))
	if m.mode != modeInput {
		t.Errorf("mode = %d, want modeInput", m.mode)
	}
	if m.inputAction != inputNewList {
		t.Errorf("inputAction = %d, want inputNewList", m.inputAction)
	}
}

func TestListPickerRename(t *testing.T) {
	m := newListPickerModel()
	m = update(t, m, key('r'))
	if m.mode != modeInput {
		t.Errorf("mode = %d, want modeInput", m.mode)
	}
	if m.inputAction != inputRenameList {
		t.Errorf("inputAction = %d, want inputRenameList", m.inputAction)
	}
}

func TestListPickerDelete(t *testing.T) {
	m := newListPickerModel()
	m = update(t, m, key('d'))
	if m.mode != modeConfirm {
		t.Errorf("mode = %d, want modeConfirm", m.mode)
	}
	if m.confirmKind != confirmDeleteList {
		t.Errorf("confirmKind = %d, want confirmDeleteList", m.confirmKind)
	}
}

func TestListPickerQuickAdd(t *testing.T) {
	m := newListPickerModel()
	m = update(t, m, key('a'))
	if m.mode != modeInput {
		t.Errorf("mode = %d, want modeInput", m.mode)
	}
	if m.inputAction != inputQuickAdd {
		t.Errorf("inputAction = %d, want inputQuickAdd", m.inputAction)
	}
}

func TestWindowResizeClamp(t *testing.T) {
	m := newTestModel()
	m.itemCursor = 4 // last item in flat list

	// Shrink to tiny window.
	m = update(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	if m.itemCursor >= len(m.flat) {
		t.Errorf("cursor %d should be < flat len %d", m.itemCursor, len(m.flat))
	}
}

func TestSettingsNavigation(t *testing.T) {
	m := New()
	m.width = 80
	m.height = 24
	m.mode = modeSettings
	m.settingsCursor = 0

	m = update(t, m, key('j'))
	if m.settingsCursor != 1 {
		t.Errorf("cursor = %d, want 1", m.settingsCursor)
	}

	m = update(t, m, key('k'))
	if m.settingsCursor != 0 {
		t.Errorf("cursor = %d, want 0", m.settingsCursor)
	}
}

func TestSettingsExit(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.Msg
	}{
		{"esc exits", specialKey(tea.KeyEscape)},
		{"q exits", key('q')},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			m.width = 80
			m.height = 24
			m.mode = modeSettings
			m = update(t, m, tt.msg)
			if m.mode != modeListPicker {
				t.Errorf("mode = %d, want modeListPicker", m.mode)
			}
		})
	}
}

func TestConfirmYes(t *testing.T) {
	m := newTestModel()
	m.prevMode = modeItems
	m.mode = modeConfirm
	m.confirmKind = confirmDeleteItem
	m.confirmItemPath = []int{1} // "Second"

	m = update(t, m, key('y'))
	if m.mode != modeItems {
		t.Errorf("mode = %d, want modeItems", m.mode)
	}
	// "Second" should be removed.
	for _, item := range m.list.Items {
		if item.Text == "Second" {
			t.Error("Second should have been deleted")
		}
	}
}

func TestConfirmNo(t *testing.T) {
	m := newTestModel()
	m.prevMode = modeItems
	m.mode = modeConfirm
	m.confirmKind = confirmDeleteItem
	m.confirmItemPath = []int{1}
	before := len(m.list.Items)

	m = update(t, m, key('n'))
	if m.mode != modeItems {
		t.Errorf("mode = %d, want modeItems", m.mode)
	}
	if len(m.list.Items) != before {
		t.Error("nothing should be deleted on 'n'")
	}
}

func TestConfirmEsc(t *testing.T) {
	m := newTestModel()
	m.prevMode = modeItems
	m.mode = modeConfirm
	m.confirmKind = confirmDeleteItem
	m.confirmItemPath = []int{1}

	m = update(t, m, specialKey(tea.KeyEscape))
	if m.mode != modeItems {
		t.Errorf("mode = %d, want modeItems", m.mode)
	}
}

func TestSearchFilter(t *testing.T) {
	m := newTestModel()
	m.mode = modeSearch
	m.searchQuery = "child"
	m.rebuildFlat()

	// Should only show Child A and Child B.
	if len(m.flat) != 2 {
		t.Errorf("expected 2 search results, got %d", len(m.flat))
	}
}

func TestSearchEscClears(t *testing.T) {
	m := newTestModel()
	m.mode = modeSearch
	m.searchQuery = "child"
	m.rebuildFlat()

	m = update(t, m, specialKey(tea.KeyEscape))
	if m.mode != modeItems {
		t.Errorf("mode = %d, want modeItems", m.mode)
	}
	if m.searchQuery != "" {
		t.Errorf("searchQuery should be empty, got %q", m.searchQuery)
	}
}

func TestHighlightMatch(t *testing.T) {
	tests := []struct {
		text, query string
		hasMatch    bool
	}{
		{"Hello World", "world", true},
		{"Hello World", "xyz", false},
		{"Hello World", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := highlightMatch(tt.text, tt.query)
			if tt.hasMatch && result == tt.text {
				t.Error("expected highlighted output to differ from input")
			}
			if !tt.hasMatch && result != tt.text {
				t.Error("expected no change for non-match")
			}
		})
	}
}

func TestPathKey(t *testing.T) {
	tests := []struct {
		path []int
		want string
	}{
		{[]int{0}, "0"},
		{[]int{1, 2}, "1,2"},
		{[]int{0, 1, 3}, "0,1,3"},
	}

	for _, tt := range tests {
		got := pathKey(tt.path)
		if got != tt.want {
			t.Errorf("pathKey(%v) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestBreadcrumb(t *testing.T) {
	items := []*todo.Item{
		{Text: "Parent", Children: []*todo.Item{
			{Text: "Child", Children: []*todo.Item{
				{Text: "Grandchild"},
			}},
		}},
	}

	tests := []struct {
		path []int
		want string
	}{
		{[]int{0}, ""},
		{[]int{0, 0}, "Parent"},
		{[]int{0, 0, 0}, "Parent › Child"},
	}

	for _, tt := range tests {
		got := breadcrumb(items, tt.path)
		if got != tt.want {
			t.Errorf("breadcrumb(%v) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestSubtreeCount(t *testing.T) {
	item := &todo.Item{
		Children: []*todo.Item{
			{Done: true},
			{Done: false, Children: []*todo.Item{
				{Done: true},
				{Done: true},
			}},
		},
	}

	total, done := subtreeCount(item)
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if done != 3 {
		t.Errorf("done = %d, want 3", done)
	}
}

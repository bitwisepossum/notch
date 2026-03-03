package todo

import (
	"testing"
	"time"
)

func newTestList() *List {
	return &List{
		Name: "test",
		Items: []*Item{
			{Text: "First", Children: []*Item{
				{Text: "Child A"},
				{Text: "Child B", Done: true},
			}},
			{Text: "Second"},
		},
	}
}

func TestAdd_TopLevel(t *testing.T) {
	l := &List{Name: "t"}
	l.Add(nil, "Item 1")
	l.Add(nil, "Item 2")
	if len(l.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(l.Items))
	}
	if l.Items[0].Text != "Item 1" {
		t.Errorf("expected 'Item 1', got %q", l.Items[0].Text)
	}
}

func TestAdd_Nested(t *testing.T) {
	l := newTestList()
	l.Add([]int{0}, "Child C")
	if len(l.Items[0].Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(l.Items[0].Children))
	}
	if l.Items[0].Children[2].Text != "Child C" {
		t.Errorf("expected 'Child C', got %q", l.Items[0].Children[2].Text)
	}
}

func TestRemove(t *testing.T) {
	l := newTestList()
	if err := l.Remove([]int{0, 0}); err != nil {
		t.Fatal(err)
	}
	if len(l.Items[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(l.Items[0].Children))
	}
	if l.Items[0].Children[0].Text != "Child B" {
		t.Errorf("expected 'Child B', got %q", l.Items[0].Children[0].Text)
	}
}

func TestRemove_TopLevel(t *testing.T) {
	l := newTestList()
	if err := l.Remove([]int{1}); err != nil {
		t.Fatal(err)
	}
	if len(l.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(l.Items))
	}
}

func TestRemove_InvalidPath(t *testing.T) {
	l := newTestList()
	if err := l.Remove([]int{5}); err == nil {
		t.Fatal("expected error for out-of-range path")
	}
	if err := l.Remove(nil); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestEdit(t *testing.T) {
	l := newTestList()
	if err := l.Edit([]int{0}, "Updated"); err != nil {
		t.Fatal(err)
	}
	if l.Items[0].Text != "Updated" {
		t.Errorf("expected 'Updated', got %q", l.Items[0].Text)
	}
}

func TestEdit_InvalidPath(t *testing.T) {
	l := newTestList()
	if err := l.Edit([]int{9}, "x"); err == nil {
		t.Fatal("expected error")
	}
}

func TestToggle(t *testing.T) {
	l := newTestList()
	if err := l.Toggle([]int{0}); err != nil {
		t.Fatal(err)
	}
	if !l.Items[0].Done {
		t.Error("expected Done=true after toggle")
	}
	if err := l.Toggle([]int{0}); err != nil {
		t.Fatal(err)
	}
	if l.Items[0].Done {
		t.Error("expected Done=false after second toggle")
	}
}

func TestToggle_AlreadyDone(t *testing.T) {
	l := newTestList()
	// Child B starts Done=true
	if err := l.Toggle([]int{0, 1}); err != nil {
		t.Fatal(err)
	}
	if l.Items[0].Children[1].Done {
		t.Error("expected Done=false after toggling completed item")
	}
}

func TestMove_Reorder(t *testing.T) {
	l := newTestList()
	// Move "Second" (index 1) to index 0
	if err := l.Move([]int{1}, []int{0}); err != nil {
		t.Fatal(err)
	}
	if l.Items[0].Text != "Second" {
		t.Errorf("expected 'Second' at 0, got %q", l.Items[0].Text)
	}
	if l.Items[1].Text != "First" {
		t.Errorf("expected 'First' at 1, got %q", l.Items[1].Text)
	}
}

func TestMove_Reparent(t *testing.T) {
	l := newTestList()
	// Move "Second" (top-level index 1) to become child of "First" at position 0
	if err := l.Move([]int{1}, []int{0, 0}); err != nil {
		t.Fatal(err)
	}
	if len(l.Items) != 1 {
		t.Fatalf("expected 1 top-level item, got %d", len(l.Items))
	}
	if l.Items[0].Children[0].Text != "Second" {
		t.Errorf("expected 'Second' as first child, got %q", l.Items[0].Children[0].Text)
	}
}

func TestMove_InvalidFrom(t *testing.T) {
	l := newTestList()
	if err := l.Move([]int{9}, []int{0}); err == nil {
		t.Fatal("expected error for invalid from path")
	}
}

func TestMove_EmptyTo(t *testing.T) {
	l := newTestList()
	if err := l.Move([]int{0}, nil); err == nil {
		t.Fatal("expected error for empty to path")
	}
}

func TestSearch_Match(t *testing.T) {
	l := newTestList()
	results := l.Search("child")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Item.Text != "Child A" {
		t.Errorf("expected 'Child A', got %q", results[0].Item.Text)
	}
	if results[1].Item.Text != "Child B" {
		t.Errorf("expected 'Child B', got %q", results[1].Item.Text)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	l := newTestList()
	results := l.Search("FIRST")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Item.Text != "First" {
		t.Errorf("expected 'First', got %q", results[0].Item.Text)
	}
}

func TestSearch_Path(t *testing.T) {
	l := newTestList()
	results := l.Search("Child B")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	want := []int{0, 1}
	got := results[0].Path
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("expected path %v, got %v", want, got)
	}
}

func TestSearch_NoMatch(t *testing.T) {
	l := newTestList()
	results := l.Search("zzz")
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	l := newTestList()
	// empty query matches all 4 items (First, Child A, Child B, Second)
	results := l.Search("")
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}
}

func TestClone_DeepCopy(t *testing.T) {
	l := newTestList()
	c := l.Clone()

	if c.Name != l.Name {
		t.Errorf("name: got %q, want %q", c.Name, l.Name)
	}
	if len(c.Items) != len(l.Items) {
		t.Fatalf("items: got %d, want %d", len(c.Items), len(l.Items))
	}
	if c.Items[0].Text != "First" {
		t.Errorf("got %q, want 'First'", c.Items[0].Text)
	}
	if len(c.Items[0].Children) != 2 {
		t.Fatalf("children: got %d, want 2", len(c.Items[0].Children))
	}
	if !c.Items[0].Children[1].Done {
		t.Error("expected Child B clone to be Done")
	}

	// Mutation isolation: changing clone does not affect original.
	c.Items[0].Text = "Modified"
	c.Items[0].Children[0].Text = "Modified Child"
	if l.Items[0].Text != "First" {
		t.Error("original mutated after clone change")
	}
	if l.Items[0].Children[0].Text != "Child A" {
		t.Error("original child mutated after clone change")
	}
}

func TestClone_Empty(t *testing.T) {
	l := &List{Name: "empty"}
	c := l.Clone()
	if c.Name != "empty" || len(c.Items) != 0 {
		t.Error("empty clone mismatch")
	}
}

func TestHash_Deterministic(t *testing.T) {
	l := newTestList()
	h1, h2 := l.Hash(), l.Hash()
	if h1 != h2 {
		t.Error("hash is not deterministic")
	}
}

func TestHash_ChangesOnTextEdit(t *testing.T) {
	l := newTestList()
	before := l.Hash()
	_ = l.Edit([]int{0}, "Changed")
	if l.Hash() == before {
		t.Error("hash did not change after editing item text")
	}
}

func TestHash_ChangesOnToggle(t *testing.T) {
	l := newTestList()
	before := l.Hash()
	_ = l.Toggle([]int{0})
	if l.Hash() == before {
		t.Error("hash did not change after toggling done state")
	}
}

func TestHash_ChangesOnStructure(t *testing.T) {
	l := newTestList()
	before := l.Hash()
	l.Add(nil, "New item")
	if l.Hash() == before {
		t.Error("hash did not change after adding an item")
	}
}

func TestHash_EmptyList(t *testing.T) {
	l := &List{Name: "empty"}
	h := l.Hash()
	if h == "" {
		t.Error("hash of empty list should not be empty string")
	}
	// A second empty list should produce the same hash regardless of name.
	l2 := &List{Name: "other"}
	if l.Hash() != l2.Hash() {
		t.Error("two empty lists should have the same hash")
	}
}

func TestSetDeadline(t *testing.T) {
	l := newTestList()
	d := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	if err := l.SetDeadline([]int{0}, d); err != nil {
		t.Fatal(err)
	}
	if !l.Items[0].Deadline.Equal(d) {
		t.Errorf("expected deadline %v, got %v", d, l.Items[0].Deadline)
	}

	// Clear the deadline.
	if err := l.SetDeadline([]int{0}, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if !l.Items[0].Deadline.IsZero() {
		t.Error("expected deadline to be cleared")
	}
}

func TestSetDeadline_InvalidPath(t *testing.T) {
	l := newTestList()
	if err := l.SetDeadline([]int{9}, time.Now()); err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestHash_ChangesOnDeadline(t *testing.T) {
	l := newTestList()
	before := l.Hash()
	_ = l.SetDeadline([]int{0}, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	if l.Hash() == before {
		t.Error("hash did not change after setting deadline")
	}
}

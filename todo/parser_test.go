package todo

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRoundTrip_Simple(t *testing.T) {
	input := `- [ ] First
- [x] Second
- [ ] Third
`
	items, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[1].Done != true {
		t.Error("expected second item to be done")
	}

	var buf bytes.Buffer
	if err := Write(&buf, items); err != nil {
		t.Fatal(err)
	}
	if buf.String() != input {
		t.Errorf("round-trip mismatch:\ngot:\n%s\nwant:\n%s", buf.String(), input)
	}
}

func TestRoundTrip_Nested(t *testing.T) {
	input := `- [ ] Project Alpha
  - [ ] Design phase
    - [x] Wireframes
    - [ ] Mockups
  - [ ] Development
    - [ ] Backend API
- [x] Team standup
`
	items, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 top-level items, got %d", len(items))
	}
	if len(items[0].Children) != 2 {
		t.Fatalf("expected 2 children under 'Project Alpha', got %d", len(items[0].Children))
	}
	if len(items[0].Children[0].Children) != 2 {
		t.Fatalf("expected 2 grandchildren under 'Design phase', got %d", len(items[0].Children[0].Children))
	}
	if !items[0].Children[0].Children[0].Done {
		t.Error("expected 'Wireframes' to be done")
	}

	var buf bytes.Buffer
	if err := Write(&buf, items); err != nil {
		t.Fatal(err)
	}
	if buf.String() != input {
		t.Errorf("round-trip mismatch:\ngot:\n%s\nwant:\n%s", buf.String(), input)
	}
}

func TestParse_EmptyInput(t *testing.T) {
	items, err := Parse(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestParse_HeadingsAndBlanks(t *testing.T) {
	input := `# My List

- [ ] Item one

# Section Two

- [x] Item two
`
	items, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Text != "Item one" {
		t.Errorf("expected 'Item one', got %q", items[0].Text)
	}
	if items[1].Text != "Item two" {
		t.Errorf("expected 'Item two', got %q", items[1].Text)
	}
}

func TestParse_DeeplyNested(t *testing.T) {
	input := `- [ ] Level 0
  - [ ] Level 1
    - [ ] Level 2
      - [ ] Level 3
        - [x] Level 4
`
	items, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	// Walk down the tree
	cur := items
	for depth := 0; depth <= 4; depth++ {
		if len(cur) != 1 {
			t.Fatalf("depth %d: expected 1 item, got %d", depth, len(cur))
		}
		expected := "Level " + string(rune('0'+depth))
		if cur[0].Text != expected {
			t.Errorf("depth %d: expected %q, got %q", depth, expected, cur[0].Text)
		}
		cur = cur[0].Children
	}
}

func TestParse_SpecialCharacters(t *testing.T) {
	input := "- [ ] Item with [brackets] and (parens)\n- [x] Item with `backticks` & ampersand\n"
	items, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Text != "Item with [brackets] and (parens)" {
		t.Errorf("unexpected text: %q", items[0].Text)
	}
	if items[1].Text != "Item with `backticks` & ampersand" {
		t.Errorf("unexpected text: %q", items[1].Text)
	}

	// Round-trip
	var buf bytes.Buffer
	if err := Write(&buf, items); err != nil {
		t.Fatal(err)
	}
	if buf.String() != input {
		t.Errorf("round-trip mismatch:\ngot:\n%s\nwant:\n%s", buf.String(), input)
	}
}

func TestWrite_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "" {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func TestParse_MalformedIndent(t *testing.T) {
	// Child indented too deep (4 spaces when parent is at 0) — should clamp
	input := `- [ ] Parent
        - [ ] Way too deep
`
	items, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 top-level item, got %d", len(items))
	}
	// The over-indented item should be clamped as a child of Parent
	if len(items[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(items[0].Children))
	}
}

func TestRoundTrip_Deadline(t *testing.T) {
	input := "- [ ] Buy groceries @2025-12-31\n- [x] Done task @2024-01-15\n"
	items, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Text != "Buy groceries" {
		t.Errorf("expected text %q, got %q", "Buy groceries", items[0].Text)
	}
	want := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	if !items[0].Deadline.Equal(want) {
		t.Errorf("expected deadline %v, got %v", want, items[0].Deadline)
	}
	if items[1].Text != "Done task" {
		t.Errorf("expected text %q, got %q", "Done task", items[1].Text)
	}

	// Round-trip: written form should match input.
	var buf bytes.Buffer
	if err := Write(&buf, items); err != nil {
		t.Fatal(err)
	}
	if buf.String() != input {
		t.Errorf("round-trip mismatch:\ngot:\n%s\nwant:\n%s", buf.String(), input)
	}
}

func TestParse_NoDeadline(t *testing.T) {
	input := "- [ ] Plain item\n"
	items, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if !items[0].Deadline.IsZero() {
		t.Errorf("expected zero deadline, got %v", items[0].Deadline)
	}
}

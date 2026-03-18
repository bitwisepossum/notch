package todo

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestRoundTrip_Simple(t *testing.T) {
	input := `- [ ] First
- [x] Second
- [ ] Third
`
	items, _, err := Parse(strings.NewReader(input))
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
	items, _, err := Parse(strings.NewReader(input))
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
	items, _, err := Parse(strings.NewReader(""))
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
	items, _, err := Parse(strings.NewReader(input))
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
	items, _, err := Parse(strings.NewReader(input))
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
	items, _, err := Parse(strings.NewReader(input))
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
	items, _, err := Parse(strings.NewReader(input))
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
	items, _, err := Parse(strings.NewReader(input))
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
	items, _, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if !items[0].Deadline.IsZero() {
		t.Errorf("expected zero deadline, got %v", items[0].Deadline)
	}
}

func TestParse_LegacyObsidianDeadline(t *testing.T) {
	input := "- [ ] Task with legacy deadline 📅 2025-06-15\n"
	items, _, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Text != "Task with legacy deadline" {
		t.Errorf("expected text %q, got %q", "Task with legacy deadline", items[0].Text)
	}
	want := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	if !items[0].Deadline.Equal(want) {
		t.Errorf("expected deadline %v, got %v", want, items[0].Deadline)
	}

	// Round-trip should convert to @ format.
	var buf bytes.Buffer
	if err := Write(&buf, items); err != nil {
		t.Fatal(err)
	}
	expected := "- [ ] Task with legacy deadline @2025-06-15\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestParse_UppercaseX(t *testing.T) {
	input := "- [X] Done with uppercase\n"
	items, _, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if !items[0].Done {
		t.Error("expected Done=true for [X]")
	}
	if items[0].Text != "Done with uppercase" {
		t.Errorf("expected text %q, got %q", "Done with uppercase", items[0].Text)
	}
}

func TestParse_NestedDeadlines(t *testing.T) {
	input := `- [ ] Parent @2025-01-01
  - [ ] Child @2025-06-15
`
	items, _, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	wantParent := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if !items[0].Deadline.Equal(wantParent) {
		t.Errorf("parent deadline: got %v, want %v", items[0].Deadline, wantParent)
	}
	wantChild := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	if !items[0].Children[0].Deadline.Equal(wantChild) {
		t.Errorf("child deadline: got %v, want %v", items[0].Children[0].Deadline, wantChild)
	}
}

func TestParse_SuspectLines(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantItems   int
		wantSuspect []string
	}{
		{
			// Lines that look like checkboxes but don't match itemRe should be collected.
			name:        "malformed lines collected",
			input:       "- [ ] Valid item\n[x] Missing dash\n* [x] Wrong bullet\n- [ ] Another valid\n",
			wantItems:   2,
			wantSuspect: []string{"[x] Missing dash", "* [x] Wrong bullet"},
		},
		{
			// "- [ ]" has no text — fails itemRe (.+ requires >=1 char) but matches suspectRe.
			name:        "empty checkbox text",
			input:       "- [ ]\n- [ ] Valid item\n",
			wantItems:   1,
			wantSuspect: []string{"- [ ]"},
		},
		{
			name:      "clean input produces no suspects",
			input:     "- [ ] First\n- [x] Second\n  - [ ] Child\n",
			wantItems: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, suspect, err := Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != tt.wantItems {
				t.Fatalf("expected %d items, got %d", tt.wantItems, len(items))
			}
			if !slices.Equal(suspect, tt.wantSuspect) {
				t.Errorf("suspect: got %v, want %v", suspect, tt.wantSuspect)
			}
		})
	}
}

func TestParse_InvalidDeadlineDate(t *testing.T) {
	// @9999-99-99 matches deadlineRe but time.Parse fails.
	// Current behaviour: deadline stays zero AND suffix is still stripped from text.
	input := "- [ ] Task @9999-99-99\n"
	items, _, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if !items[0].Deadline.IsZero() {
		t.Errorf("expected zero deadline for invalid date, got %v", items[0].Deadline)
	}
	if items[0].Text != "Task" {
		t.Errorf("expected text %q (suffix stripped), got %q", "Task", items[0].Text)
	}
}

package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/bitwisepossum/notch/todo"
)

// tempHome redirects DataDir to a temp directory for the duration of t.
func tempHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestCmdVersion(t *testing.T) {
	var buf bytes.Buffer
	if err := cmdVersion(&buf); err != nil {
		t.Fatal(err)
	}
	want := "notch v" + todo.Version + "\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCmdLs_Empty(t *testing.T) {
	tempHome(t)
	var buf bytes.Buffer
	if err := cmdLs(&buf); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output, got %q", buf.String())
	}
}

func TestCmdLs_WithLists(t *testing.T) {
	tempHome(t)
	todo.Save(&todo.List{Name: "alpha", Items: []*todo.Item{{Text: "a"}}})
	todo.Save(&todo.List{Name: "beta", Items: []*todo.Item{{Text: "b"}}})

	var buf bytes.Buffer
	if err := cmdLs(&buf); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
}

func TestCmdCat_Existing(t *testing.T) {
	tempHome(t)
	todo.Save(&todo.List{Name: "work", Items: []*todo.Item{
		{Text: "Task A"},
		{Text: "Task B", Done: true},
	}})

	var buf bytes.Buffer
	if err := cmdCat(&buf, []string{"work"}); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "- [ ] Task A") {
		t.Errorf("missing open item: %q", got)
	}
	if !strings.Contains(got, "- [x] Task B") {
		t.Errorf("missing done item: %q", got)
	}
}

func TestCmdCat_Missing(t *testing.T) {
	tempHome(t)
	err := cmdCat(&bytes.Buffer{}, []string{"nope"})
	if err == nil {
		t.Fatal("expected error for missing list")
	}
}

func TestCmdCat_BadArgs(t *testing.T) {
	err := cmdCat(&bytes.Buffer{}, nil)
	if err == nil {
		t.Fatal("expected error for no args")
	}
}

func TestCmdAdd_NewList(t *testing.T) {
	tempHome(t)
	if err := cmdAdd([]string{"shopping", "Milk"}); err != nil {
		t.Fatal(err)
	}
	list, err := todo.Load("shopping")
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].Text != "Milk" {
		t.Errorf("unexpected items: %v", list.Items)
	}
}

func TestCmdAdd_ExistingList(t *testing.T) {
	tempHome(t)
	todo.Save(&todo.List{Name: "chores", Items: []*todo.Item{{Text: "Sweep"}}})
	if err := cmdAdd([]string{"chores", "Mop"}); err != nil {
		t.Fatal(err)
	}
	list, err := todo.Load("chores")
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(list.Items))
	}
}

func TestCmdAdd_JoinsArgs(t *testing.T) {
	tempHome(t)
	if err := cmdAdd([]string{"misc", "Buy", "more", "milk"}); err != nil {
		t.Fatal(err)
	}
	list, _ := todo.Load("misc")
	if list.Items[0].Text != "Buy more milk" {
		t.Errorf("got %q", list.Items[0].Text)
	}
}

func TestCmdDone_Match(t *testing.T) {
	tempHome(t)
	todo.Save(&todo.List{Name: "todo", Items: []*todo.Item{
		{Text: "Buy milk"},
		{Text: "Read book"},
	}})
	if err := cmdDone([]string{"todo", "milk"}); err != nil {
		t.Fatal(err)
	}
	list, _ := todo.Load("todo")
	if !list.Items[0].Done {
		t.Error("expected first item to be done")
	}
	if list.Items[1].Done {
		t.Error("expected second item to remain open")
	}
}

func TestCmdDone_NoMatch(t *testing.T) {
	tempHome(t)
	todo.Save(&todo.List{Name: "todo", Items: []*todo.Item{{Text: "Task"}}})
	err := cmdDone([]string{"todo", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for no match")
	}
}

func TestCmdDone_AlreadyDone(t *testing.T) {
	tempHome(t)
	todo.Save(&todo.List{Name: "todo", Items: []*todo.Item{{Text: "Done task", Done: true}}})
	err := cmdDone([]string{"todo", "Done"})
	if err == nil {
		t.Fatal("expected error for already done")
	}
}

func TestCmdRm_Force(t *testing.T) {
	tempHome(t)
	todo.Save(&todo.List{Name: "trash", Items: []*todo.Item{{Text: "x"}}})
	if err := cmdRm(strings.NewReader(""), []string{"-f", "trash"}); err != nil {
		t.Fatal(err)
	}
	names, _ := todo.ListAll()
	for _, n := range names {
		if n == "trash" {
			t.Error("list should have been deleted")
		}
	}
}

func TestCmdRm_Confirm(t *testing.T) {
	tempHome(t)
	todo.Save(&todo.List{Name: "keep", Items: []*todo.Item{{Text: "x"}}})
	// Simulate typing "n"
	if err := cmdRm(strings.NewReader("n\n"), []string{"keep"}); err != nil {
		t.Fatal(err)
	}
	names, _ := todo.ListAll()
	found := false
	for _, n := range names {
		if n == "keep" {
			found = true
		}
	}
	if !found {
		t.Error("list should not have been deleted after declining")
	}
}

func TestCmdRm_ConfirmYes(t *testing.T) {
	tempHome(t)
	todo.Save(&todo.List{Name: "bye", Items: []*todo.Item{{Text: "x"}}})
	if err := cmdRm(strings.NewReader("y\n"), []string{"bye"}); err != nil {
		t.Fatal(err)
	}
	// Verify it loads with not-exist error.
	_, err := todo.Load("bye")
	if !os.IsNotExist(err) {
		t.Error("list should have been deleted after confirming")
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	err := run([]string{"bogus"}, &bytes.Buffer{}, strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

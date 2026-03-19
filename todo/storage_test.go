package todo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSave_CreatesFile(t *testing.T) {
	dir, _ := tempHome(t)
	list := &List{Name: "grocery", Items: []*Item{{Text: "Milk"}}}
	if err := Save(list); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "grocery.md"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "- [ ] Milk\n" {
		t.Errorf("got %q", got)
	}
}

func TestSave_Overwrites(t *testing.T) {
	dir, _ := tempHome(t)
	list := &List{Name: "work", Items: []*Item{{Text: "First"}}}
	if err := Save(list); err != nil {
		t.Fatal(err)
	}
	list.Items[0].Text = "Second"
	if err := Save(list); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "work.md"))
	if got := string(data); got != "- [ ] Second\n" {
		t.Errorf("got %q, want second version", got)
	}
}

func TestSave_NoTmpRemains(t *testing.T) {
	dir, _ := tempHome(t)
	list := &List{Name: "test", Items: []*Item{{Text: "Hello"}}}
	if err := Save(list); err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(dir, "test.md.tmp")
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error("expected .tmp to be cleaned up")
	}
}

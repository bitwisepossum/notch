package todo

import (
	"os"
	"path/filepath"
	"strings"
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

func TestSave_CreatesBak(t *testing.T) {
	dir, _ := tempHome(t)
	list := &List{Name: "work", Items: []*Item{{Text: "First"}}}
	if err := Save(list); err != nil {
		t.Fatal(err)
	}
	list.Items[0].Text = "Second"
	if err := Save(list); err != nil {
		t.Fatal(err)
	}
	bak, err := os.ReadFile(filepath.Join(dir, "work.md.bak"))
	if err != nil {
		t.Fatal("expected .bak file:", err)
	}
	if got := string(bak); got != "- [ ] First\n" {
		t.Errorf("bak content = %q, want first version", got)
	}
	cur, _ := os.ReadFile(filepath.Join(dir, "work.md"))
	if got := string(cur); got != "- [ ] Second\n" {
		t.Errorf("current content = %q, want second version", got)
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

func TestSaveSettings_CreatesBak(t *testing.T) {
	dir, _ := tempHome(t)
	s := Settings{LogLevel: "minimal"}
	if err := SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	s.LogLevel = "full"
	if err := SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	bak, err := os.ReadFile(filepath.Join(dir, "settings.json.bak"))
	if err != nil {
		t.Fatal("expected settings .bak:", err)
	}
	if !strings.Contains(string(bak), "minimal") {
		t.Error("bak should contain first version")
	}
}

func TestListAll_ExcludesBakAndTmp(t *testing.T) {
	dir, _ := tempHome(t)
	// Create a real list.
	list := &List{Name: "real", Items: []*Item{{Text: "Item"}}}
	if err := Save(list); err != nil {
		t.Fatal(err)
	}
	// Plant .bak and .tmp files.
	os.WriteFile(filepath.Join(dir, "stale.md.bak"), []byte("bak"), 0o644)
	os.WriteFile(filepath.Join(dir, "partial.md.tmp"), []byte("tmp"), 0o644)

	names, err := ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "real" {
		t.Errorf("ListAll() = %v, want [real]", names)
	}
}

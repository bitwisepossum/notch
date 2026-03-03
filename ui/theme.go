package ui

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitwisepossum/notch/todo"
)

// Theme defines the color palette used by the TUI.
type Theme struct {
	Key       string // filename stem; empty for the built-in default (not in JSON)
	Name      string `json:"name"`
	BgSelect  string `json:"bg_select"`
	Muted     string `json:"muted"`
	Primary   string `json:"primary"`
	Accent    string `json:"accent"`
	Danger    string `json:"danger"`
	Separator string `json:"separator"`
	Border    string `json:"border"`
	Done      string `json:"done"`
}

// DefaultTheme is the built-in Nokia LCD-inspired green palette.
var DefaultTheme = Theme{
	Key:       "",
	Name:      "Default",
	BgSelect:  "#1A2508",
	Muted:     "#556820",
	Primary:   "#9BB030",
	Accent:    "#D0E040",
	Danger:    "#C86050",
	Separator: "#405010",
	Border:    "#708830",
	Done:      "#3A4818",
}

const themesSubdir = "themes"

// LoadThemes scans <DataDir>/themes/ for *.json files and returns all themes.
// The built-in DefaultTheme is always the first entry.
func LoadThemes() []Theme {
	themes := []Theme{DefaultTheme}

	dir, err := todo.DataDir()
	if err != nil {
		return themes
	}

	td := filepath.Join(dir, themesSubdir)
	if err := os.MkdirAll(td, 0o750); err != nil {
		return themes
	}

	entries, err := os.ReadDir(td)
	if err != nil {
		return themes
	}

	root, err := os.OpenRoot(td)
	if err != nil {
		return themes
	}
	defer root.Close()

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fname := e.Name()
		if !strings.HasSuffix(fname, ".json") {
			continue
		}
		f, err := root.Open(fname)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			continue
		}
		var t Theme
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}
		t.Key = strings.TrimSuffix(fname, ".json")
		if t.Name == "" {
			t.Name = t.Key
		}
		themes = append(themes, t)
	}
	return themes
}

package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
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
	// Optional UI characters; omit to use built-in defaults.
	CheckDone string `json:"check_done,omitempty"` // default: "✓"
	CheckOpen string `json:"check_open,omitempty"` // default: "○"
	BarFilled string `json:"bar_filled,omitempty"` // default: "━"
	BarEmpty  string `json:"bar_empty,omitempty"`  // default: "─"
	// Error is set at load time if the file is malformed; never persisted.
	Error string `json:"-"`
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
	CheckDone: "✓",
	CheckOpen: "○",
	BarFilled: "━",
	BarEmpty:  "─",
}

const themesSubdir = "themes"

// isValidHex reports whether s is a valid CSS hex color (#RGB or #RRGGBB).
func isValidHex(s string) bool {
	if len(s) != 4 && len(s) != 7 {
		return false
	}
	if s[0] != '#' {
		return false
	}
	for _, c := range s[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// validate checks that all required color fields are valid hex and optional
// character fields are single display characters. Returns the first error found.
func (t Theme) validate() error {
	required := []struct {
		name string
		val  string
	}{
		{"bg_select", t.BgSelect},
		{"muted", t.Muted},
		{"primary", t.Primary},
		{"accent", t.Accent},
		{"danger", t.Danger},
		{"separator", t.Separator},
		{"border", t.Border},
		{"done", t.Done},
	}
	for _, f := range required {
		if !isValidHex(f.val) {
			return fmt.Errorf("%s: invalid hex color %q", f.name, f.val)
		}
	}
	chars := []struct {
		name string
		val  string
	}{
		{"check_done", t.CheckDone},
		{"check_open", t.CheckOpen},
		{"bar_filled", t.BarFilled},
		{"bar_empty", t.BarEmpty},
	}
	for _, c := range chars {
		if c.val != "" && lipgloss.Width(c.val) != 1 {
			return fmt.Errorf("%s: must be a single display character", c.name)
		}
	}
	return nil
}

// LoadThemes scans <DataDir>/themes/ for *.json files and returns all themes.
// The built-in DefaultTheme is always the first entry. Malformed files are
// included with their Error field set so the settings UI can surface them.
func LoadThemes() []Theme {
	themes := []Theme{DefaultTheme}

	dir, err := todo.DataDir()
	if err != nil {
		todo.LogError("themes data dir", slog.String("err", err.Error()))
		return themes
	}

	td := filepath.Join(dir, themesSubdir)
	if err := os.MkdirAll(td, 0o750); err != nil {
		todo.LogError("create themes dir", slog.String("err", err.Error()))
		return themes
	}

	entries, err := os.ReadDir(td)
	if err != nil {
		todo.LogError("read themes dir", slog.String("err", err.Error()))
		return themes
	}

	root, err := os.OpenRoot(td)
	if err != nil {
		todo.LogError("open themes dir", slog.String("err", err.Error()))
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
		key := strings.TrimSuffix(fname, ".json")
		f, err := root.Open(fname)
		if err != nil {
			todo.LogError("open theme", slog.String("file", fname), slog.String("err", err.Error()))
			continue
		}
		data, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			todo.LogError("read theme", slog.String("file", fname), slog.String("err", err.Error()))
			continue
		}
		var t Theme
		if err := json.Unmarshal(data, &t); err != nil {
			todo.LogError("parse theme", slog.String("file", fname), slog.String("err", err.Error()))
			themes = append(themes, Theme{
				Key:   key,
				Name:  key,
				Error: "invalid JSON: " + err.Error(),
			})
			continue
		}
		t.Key = key
		if t.Name == "" {
			t.Name = key
		}
		if err := t.validate(); err != nil {
			todo.LogError("invalid theme", slog.String("file", fname), slog.String("err", err.Error()))
			t.Error = err.Error()
		}
		themes = append(themes, t)
	}
	return themes
}

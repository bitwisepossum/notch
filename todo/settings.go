package todo

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
)

// SavedFolds holds the persisted fold state for one list.
type SavedFolds struct {
	Hash  string   `json:"hash"`
	Paths []string `json:"paths"` // index-based path keys ("0", "0,2", etc.)
}

// Settings holds user-configurable application settings.
type Settings struct {
	CustomDataDir string                `json:"custom_data_dir,omitempty"`
	ActiveTheme   string                `json:"active_theme,omitempty"` // theme Key; empty = built-in default
	FoldState     map[string]SavedFolds `json:"fold_state,omitempty"`   // list name → saved folds
	CascadeDone   bool                  `json:"cascade_done,omitempty"` // marking done also marks all children
	// DeadlineFormat controls how deadlines are displayed and parsed in the UI.
	// Deadlines are still persisted to list files as YYYY-MM-DD for portability.
	// Empty means default (YYYY-MM-DD).
	DeadlineFormat string `json:"deadline_format,omitempty"`
	// ShowHelp controls sidebar visibility. Nil means "never set" (show on first start).
	ShowHelp *bool `json:"show_help,omitempty"`
	// LogLevel controls file logging. Empty or "off" disables logging.
	// "minimal" logs errors only; "full" logs errors and user actions.
	LogLevel string `json:"log_level,omitempty"`
}

const settingsFile = "settings.json"

// LoadSettings reads settings from the default app data directory.
// Returns zero-value Settings if the file does not exist.
func LoadSettings() (Settings, error) {
	dir, err := DataDir()
	if err != nil {
		return Settings{}, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return Settings{}, err
	}
	defer root.Close()
	f, err := root.Open(settingsFile)
	if os.IsNotExist(err) {
		return Settings{}, nil
	}
	if err != nil {
		return Settings{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		LogError("read settings", slog.String("err", err.Error()))
		return Settings{}, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		LogError("parse settings", slog.String("err", err.Error()))
		return Settings{}, err
	}
	return s, nil
}

// SaveSettings writes settings to the default app data directory.
// The write is atomic: data goes to a temp file, then renamed over the target.
func SaveSettings(s Settings) error {
	dir, err := DataDir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		LogError("marshal settings", slog.String("err", err.Error()))
		return err
	}
	data = append(data, '\n')
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := atomicWrite(root, settingsFile, data, 0o600); err != nil {
		LogError("write settings", slog.String("err", err.Error()))
		return err
	}
	return nil
}

// ListDir returns the directory where list .md files are stored.
// Uses CustomDataDir from settings if set and accessible; otherwise falls back to DataDir.
func ListDir() (string, error) {
	s, err := LoadSettings()
	if err == nil && s.CustomDataDir != "" {
		if mkErr := os.MkdirAll(s.CustomDataDir, 0o750); mkErr == nil {
			return s.CustomDataDir, nil
		}
	}
	return DataDir()
}

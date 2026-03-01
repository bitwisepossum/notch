package todo

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings holds user-configurable application settings.
type Settings struct {
	CustomDataDir string `json:"custom_data_dir,omitempty"`
	ActiveTheme   string `json:"active_theme,omitempty"` // theme Key; empty = built-in default
}

const settingsFile = "settings.json"

// LoadSettings reads settings from the default app data directory.
// Returns zero-value Settings if the file does not exist.
func LoadSettings() (Settings, error) {
	dir, err := DataDir()
	if err != nil {
		return Settings{}, err
	}
	data, err := os.ReadFile(filepath.Join(dir, settingsFile))
	if os.IsNotExist(err) {
		return Settings{}, nil
	}
	if err != nil {
		return Settings{}, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, err
	}
	return s, nil
}

// SaveSettings writes settings to the default app data directory.
func SaveSettings(s Settings) error {
	dir, err := DataDir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, settingsFile), data, 0o640)
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

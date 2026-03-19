package ui

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"

	"github.com/bitwisepossum/notch/todo"
)

// SavedFolds holds the persisted fold state for one list.
type SavedFolds struct {
	Hash  string   `json:"hash"`
	Paths []string `json:"paths"` // index-based path keys ("0", "0,2", etc.)
}

// UIState holds view-layer state that is persisted across sessions but does
// not belong in the backend Settings struct.
type UIState struct {
	FoldState map[string]SavedFolds `json:"fold_state,omitempty"` // list name → saved folds
	// ShowHelp controls sidebar visibility. Nil means "never set" (show on first start).
	ShowHelp *bool `json:"show_help,omitempty"`
}

const uiStateFile = "ui_state.json"

// loadUIState reads UI state from the app data directory.
// Returns zero-value UIState if the file does not exist.
func loadUIState() (UIState, error) {
	dir, err := todo.DataDir()
	if err != nil {
		return UIState{}, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return UIState{}, err
	}
	defer root.Close()
	f, err := root.Open(uiStateFile)
	if os.IsNotExist(err) {
		return UIState{}, nil
	}
	if err != nil {
		return UIState{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		todo.LogError("read ui state", slog.String("err", err.Error()))
		return UIState{}, err
	}
	var s UIState
	if err := json.Unmarshal(data, &s); err != nil {
		todo.LogError("parse ui state", slog.String("err", err.Error()))
		return UIState{}, err
	}
	return s, nil
}

// saveUIState writes UI state to the app data directory.
// The write is atomic: data goes to a temp file, then renamed over the target.
func saveUIState(s UIState) error {
	dir, err := todo.DataDir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		todo.LogError("marshal ui state", slog.String("err", err.Error()))
		return err
	}
	data = append(data, '\n')
	return todo.AtomicWriteFile(dir, uiStateFile, data, 0o600)
}

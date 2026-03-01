package todo

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DataDir returns the storage directory for notch lists, creating it if needed.
//
// On Linux:   ~/.local/share/notch
// On macOS:   ~/Library/Application Support/notch
// On Windows: %APPDATA%\notch
func DataDir() (string, error) {
	var dir string
	switch runtime.GOOS {
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".local", "share", "notch")
	default:
		configDir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(configDir, "notch")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	return dir, nil
}

// openRoot returns an os.Root scoped to the list directory.
func openRoot() (*os.Root, error) {
	dir, err := ListDir()
	if err != nil {
		return nil, err
	}
	return os.OpenRoot(dir)
}

// ListAll returns the names of all saved lists (filenames without .md).
func ListAll() ([]string, error) {
	dir, err := ListDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".md") {
			names = append(names, strings.TrimSuffix(name, ".md"))
		}
	}
	return names, nil
}

// Load reads and parses a list from its Markdown file.
func Load(name string) (*List, error) {
	root, err := openRoot()
	if err != nil {
		return nil, err
	}
	defer root.Close()

	f, err := root.Open(name + ".md")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	items, err := Parse(f)
	if err != nil {
		return nil, err
	}
	return &List{Name: name, Items: items}, nil
}

// Save writes a list to its Markdown file, creating or overwriting it.
func Save(list *List) error {
	root, err := openRoot()
	if err != nil {
		return err
	}
	defer root.Close()

	f, err := root.Create(list.Name + ".md")
	if err != nil {
		return err
	}
	defer f.Close()

	return Write(f, list.Items)
}

// Delete removes a list's Markdown file.
func Delete(name string) error {
	root, err := openRoot()
	if err != nil {
		return err
	}
	defer root.Close()

	return root.Remove(name + ".md")
}

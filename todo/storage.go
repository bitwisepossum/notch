package todo

import (
	"bytes"
	"log/slog"
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

	items, suspect, err := Parse(f)
	if err != nil {
		LogError("parse error", slog.String("list", name), slog.String("err", err.Error()))
		return nil, err
	}
	for _, line := range suspect {
		LogError("skipped malformed checkbox line", slog.String("list", name), slog.String("line", line))
	}
	return &List{Name: name, Items: items}, nil
}

// Save writes a list to its Markdown file, creating or overwriting it.
// The write is atomic: data goes to a temp file first, then is renamed
// over the target. A .bak copy of the previous version is kept.
func Save(list *List) error {
	root, err := openRoot()
	if err != nil {
		return err
	}
	defer root.Close()

	var buf bytes.Buffer
	if err := Write(&buf, list.Items); err != nil {
		return err
	}
	return atomicWrite(root, list.Name+".md", buf.Bytes(), 0o644)
}

// atomicWrite safely writes data to name within root using a temp+rename
// pattern. A .bak copy of the previous file is kept.
func atomicWrite(root *os.Root, name string, data []byte, perm os.FileMode) error {
	tmp := name + ".tmp"

	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		root.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		root.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		root.Remove(tmp)
		return err
	}

	// Keep a backup of the previous version.
	bak := name + ".bak"
	if _, err := root.Stat(name); err == nil {
		root.Remove(bak)
		root.Rename(name, bak)
	}

	// Rename temp to target. On failure, try to restore from backup.
	if err := root.Rename(tmp, name); err != nil {
		root.Rename(bak, name)
		return err
	}
	return nil
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

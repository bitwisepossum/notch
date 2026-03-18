package todo

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Log level constants.
const (
	LogOff     = "off"
	LogMinimal = "minimal"
	LogFull    = "full"
)

const (
	logFileName = "notch.log"
	maxLogSize  = 2 << 20 // 2 MB
	pruneTarget = 1 << 20 // keep last 1 MB after pruning
)

// logger is the package-level logger singleton.
var logger struct {
	mu      sync.Mutex
	level   string
	handler *slog.Logger
	file    *os.File
}

// InitLogger opens the log file in DataDir and configures the logger.
// It should be called once on startup. Calling it again reconfigures.
func InitLogger(level string) error {
	logger.mu.Lock()
	defer logger.mu.Unlock()

	if logger.file != nil {
		_ = logger.file.Close()
		logger.file = nil
	}
	logger.level = level
	logger.handler = nil

	if level == "" || level == LogOff {
		return nil
	}

	dir, err := DataDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, logFileName)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600) // #nosec G304
	if err != nil {
		return err
	}
	logger.file = f
	h := slog.New(slog.NewTextHandler(f, nil))
	logger.handler = h

	pruneLog(path)
	h.LogAttrs(context.TODO(), slog.LevelInfo, "logger started",
		slog.String("path", SanitizePath(path)),
		slog.String("level", level),
	)
	return nil
}

// SetLogLevel reconfigures the logger level at runtime.
// If the file is already open it is reused; if switching from off it is opened.
func SetLogLevel(level string) {
	logger.mu.Lock()
	old := logger.level
	logger.mu.Unlock()

	if old == level {
		return
	}
	_ = InitLogger(level)
}

// CloseLogger flushes and closes the log file.
func CloseLogger() {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	if logger.file != nil {
		_ = logger.file.Sync()
		_ = logger.file.Close()
		logger.file = nil
	}
	logger.handler = nil
}

// ClearLog truncates the log file to zero bytes.
func ClearLog() error {
	logger.mu.Lock()
	defer logger.mu.Unlock()

	if logger.file == nil {
		// Logger is off; still truncate if file exists.
		dir, err := DataDir()
		if err != nil {
			return err
		}
		path := filepath.Join(dir, logFileName)
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0o600) // #nosec G304
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		return f.Close()
	}

	return logger.file.Truncate(0)
}

// ReadLog returns the full contents of the log file as a string.
func ReadLog() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, logFileName)
	data, err := os.ReadFile(path) // #nosec G304
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// LogSize returns the current size of the log file in bytes, or 0 if absent.
func LogSize() int64 {
	dir, err := DataDir()
	if err != nil {
		return 0
	}
	info, err := os.Stat(filepath.Join(dir, logFileName))
	if err != nil {
		return 0
	}
	return info.Size()
}

// LogError logs msg at minimal+ level (errors, IO failures, parse errors).
// attrs are optional structured key-value pairs (use slog.String, slog.Int, etc.).
func LogError(msg string, attrs ...slog.Attr) {
	logger.mu.Lock()
	h := logger.handler
	level := logger.level
	logger.mu.Unlock()

	if h == nil || level == "" || level == LogOff {
		return
	}
	h.LogAttrs(context.TODO(), slog.LevelError, msg, attrs...)
}

// LogEvent logs msg at full level only (user actions).
func LogEvent(msg string, attrs ...slog.Attr) {
	logger.mu.Lock()
	h := logger.handler
	level := logger.level
	logger.mu.Unlock()

	if h == nil || level != LogFull {
		return
	}
	h.LogAttrs(context.TODO(), slog.LevelInfo, msg, attrs...)
}

// SanitizePath replaces the user's home directory prefix with ~.
func SanitizePath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

// pruneLog truncates the log at path if it exceeds maxLogSize, keeping the
// last pruneTarget bytes aligned to the next newline boundary.
func pruneLog(path string) {
	info, err := os.Stat(path)
	if err != nil || info.Size() <= maxLogSize {
		return
	}

	f, err := os.OpenFile(path, os.O_RDWR, 0o600) // #nosec G304
	if err != nil {
		return
	}
	defer f.Close()

	size := info.Size()
	offset := max(size-pruneTarget, 0)

	// Align to the next newline after offset so we don't split an entry.
	buf := make([]byte, 4096)
	n, err := f.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return
	}
	nl := strings.Index(string(buf[:n]), "\n")
	if nl >= 0 {
		offset += int64(nl) + 1
	}

	tail := make([]byte, size-offset)
	if _, err := f.ReadAt(tail, offset); err != nil && err != io.EOF {
		return
	}
	if err := f.Truncate(0); err != nil {
		return
	}
	if _, err := f.WriteAt(tail, 0); err != nil {
		return
	}
}

package todo

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetLogger tears down the logger singleton to a clean state.
// Always call via t.Cleanup so each test starts fresh.
func resetLogger() {
	CloseLogger()
	logger.mu.Lock()
	logger.level = ""
	logger.mu.Unlock()
}

// tempHome redirects DataDir to a temp directory for the duration of t by
// overriding $HOME. It also schedules logger teardown via t.Cleanup.
// Returns (dataDir, logFilePath).
func tempHome(t *testing.T) (string, string) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	InvalidateListDir()
	dir, err := DataDir()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(resetLogger)
	return dir, filepath.Join(dir, logFileName)
}

func writeLog(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readLog(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// --- pruneLog ---

func TestPruneLog_BelowThreshold_Unchanged(t *testing.T) {
	_, logPath := tempHome(t)
	content := bytes.Repeat([]byte("x"), maxLogSize-1)
	writeLog(t, logPath, content)

	pruneLog(logPath)

	if got := readLog(t, logPath); len(got) != len(content) {
		t.Errorf("expected no pruning below threshold: size %d → %d", len(content), len(got))
	}
}

func TestPruneLog_ExactlyAtThreshold_Unchanged(t *testing.T) {
	_, logPath := tempHome(t)
	content := bytes.Repeat([]byte("x"), maxLogSize)
	writeLog(t, logPath, content)

	pruneLog(logPath)

	if got := readLog(t, logPath); len(got) != len(content) {
		t.Errorf("expected no pruning at exact threshold: size %d → %d", len(content), len(got))
	}
}

func TestPruneLog_OversizeTruncatesToZero(t *testing.T) {
	_, logPath := tempHome(t)

	// 3 MB of 80-byte lines (79 'a's + newline).
	line := []byte(strings.Repeat("a", 79) + "\n")
	var buf bytes.Buffer
	for buf.Len() < 3*(1<<20) {
		buf.Write(line)
	}
	writeLog(t, logPath, buf.Bytes())

	pruneLog(logPath)

	got := readLog(t, logPath)
	if len(got) != 0 {
		t.Errorf("expected empty file after prune, got %d bytes", len(got))
	}
}

func TestPruneLog_NonexistentFile_DoesNotPanic(t *testing.T) {
	_, logPath := tempHome(t)
	// File does not exist — must be a no-op.
	pruneLog(logPath)
}

// --- InitLogger ---

func TestInitLogger_Off_NoFileCreated(t *testing.T) {
	_, logPath := tempHome(t)

	if err := InitLogger(LogOff); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("log file should not be created when level is off")
	}
}

func TestInitLogger_EmptyLevel_NoFileCreated(t *testing.T) {
	_, logPath := tempHome(t)

	if err := InitLogger(""); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("log file should not be created when level is empty")
	}
}

func TestInitLogger_Minimal_CreatesFileWithStartEntry(t *testing.T) {
	_, logPath := tempHome(t)

	if err := InitLogger(LogMinimal); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(readLog(t, logPath)), "logger started") {
		t.Error("expected 'logger started' entry in log")
	}
}

func TestInitLogger_Full_CreatesFile(t *testing.T) {
	_, logPath := tempHome(t)

	if err := InitLogger(LogFull); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("log file not created: %v", err)
	}
}

func TestInitLogger_Reinit_FileRemainsWritable(t *testing.T) {
	_, logPath := tempHome(t)

	if err := InitLogger(LogMinimal); err != nil {
		t.Fatal(err)
	}
	if err := InitLogger(LogMinimal); err != nil {
		t.Fatal(err)
	}

	LogError("post-reinit message")

	if !strings.Contains(string(readLog(t, logPath)), "post-reinit message") {
		t.Error("expected message to appear in log after reinit")
	}
}

func TestInitLogger_PrunesOversizeFileOnStart(t *testing.T) {
	_, logPath := tempHome(t)

	// Pre-populate log with 3 MB of data so prune triggers on init.
	line := []byte(strings.Repeat("a", 79) + "\n")
	var buf bytes.Buffer
	for buf.Len() < 3*(1<<20) {
		buf.Write(line)
	}
	writeLog(t, logPath, buf.Bytes())
	origSize := buf.Len()

	if err := InitLogger(LogMinimal); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if int(info.Size()) >= origSize {
		t.Errorf("expected file to be pruned during init, size unchanged at %d", info.Size())
	}
}

// --- SetLogLevel ---

func TestSetLogLevel_SameLevel_NoReopen(t *testing.T) {
	tempHome(t)

	if err := InitLogger(LogMinimal); err != nil {
		t.Fatal(err)
	}
	logger.mu.Lock()
	f1 := logger.file
	logger.mu.Unlock()

	SetLogLevel(LogMinimal)

	logger.mu.Lock()
	f2 := logger.file
	logger.mu.Unlock()

	if f1 != f2 {
		t.Error("SetLogLevel with same level should not reopen the file")
	}
}

func TestSetLogLevel_OffToMinimal_OpensFile(t *testing.T) {
	_, logPath := tempHome(t)

	if err := InitLogger(LogOff); err != nil {
		t.Fatal(err)
	}

	SetLogLevel(LogMinimal)

	if _, err := os.Stat(logPath); err != nil {
		t.Errorf("log file should exist after switching from off to minimal: %v", err)
	}
}

func TestSetLogLevel_MinimalToOff_ClosesHandler(t *testing.T) {
	tempHome(t)

	if err := InitLogger(LogMinimal); err != nil {
		t.Fatal(err)
	}

	SetLogLevel(LogOff)

	logger.mu.Lock()
	h := logger.handler
	logger.mu.Unlock()

	if h != nil {
		t.Error("handler should be nil after switching to off")
	}
}

// --- LogError / LogEvent level filtering ---

func TestLogError_Off_WritesNothing(t *testing.T) {
	_, logPath := tempHome(t)

	if err := InitLogger(LogOff); err != nil {
		t.Fatal(err)
	}
	LogError("should not appear")

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("no log file should exist when level is off")
	}
}

func TestLogError_Minimal_WritesEntry(t *testing.T) {
	_, logPath := tempHome(t)

	if err := InitLogger(LogMinimal); err != nil {
		t.Fatal(err)
	}
	LogError("test error entry")

	if !strings.Contains(string(readLog(t, logPath)), "test error entry") {
		t.Error("expected error entry in log at minimal level")
	}
}

func TestLogEvent_Minimal_WritesNothing(t *testing.T) {
	_, logPath := tempHome(t)

	if err := InitLogger(LogMinimal); err != nil {
		t.Fatal(err)
	}
	LogEvent("event on minimal")

	if strings.Contains(string(readLog(t, logPath)), "event on minimal") {
		t.Error("LogEvent should not write at minimal level")
	}
}

func TestLogError_Full_WritesEntry(t *testing.T) {
	_, logPath := tempHome(t)

	if err := InitLogger(LogFull); err != nil {
		t.Fatal(err)
	}
	LogError("error on full level")

	if !strings.Contains(string(readLog(t, logPath)), "error on full level") {
		t.Error("expected error entry in log at full level")
	}
}

func TestLogEvent_Full_WritesEntry(t *testing.T) {
	_, logPath := tempHome(t)

	if err := InitLogger(LogFull); err != nil {
		t.Fatal(err)
	}
	LogEvent("event on full level")

	if !strings.Contains(string(readLog(t, logPath)), "event on full level") {
		t.Error("expected event entry in log at full level")
	}
}

// --- ClearLog ---

func TestClearLog_ActiveLogger_TruncatesFile(t *testing.T) {
	_, logPath := tempHome(t)

	if err := InitLogger(LogMinimal); err != nil {
		t.Fatal(err)
	}
	LogError("before clear")

	if err := ClearLog(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Errorf("expected zero-byte file after clear, got %d bytes", info.Size())
	}
}

func TestClearLog_LoggerOff_FileExists_TruncatesFile(t *testing.T) {
	_, logPath := tempHome(t)
	writeLog(t, logPath, []byte("pre-existing log content"))

	// Logger is off (never initialised in this test).
	if err := ClearLog(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Errorf("expected zero bytes after clear, got %d", info.Size())
	}
}

func TestClearLog_LoggerOff_NoFile_ReturnsNil(t *testing.T) {
	tempHome(t)

	if err := ClearLog(); err != nil {
		t.Errorf("expected no error when file is absent, got: %v", err)
	}
}

func TestClearLog_ActiveLogger_SubsequentWritesAppend(t *testing.T) {
	_, logPath := tempHome(t)

	if err := InitLogger(LogMinimal); err != nil {
		t.Fatal(err)
	}
	LogError("before clear")

	if err := ClearLog(); err != nil {
		t.Fatal(err)
	}

	LogError("after clear")

	content := string(readLog(t, logPath))
	if strings.Contains(content, "before clear") {
		t.Error("pre-clear entry should not appear after truncation")
	}
	if !strings.Contains(content, "after clear") {
		t.Error("expected post-clear entry to appear")
	}
}

// --- ReadLog ---

func TestReadLog_Content(t *testing.T) {
	_, logPath := tempHome(t)
	writeLog(t, logPath, []byte("hello log"))

	content, err := ReadLog()
	if err != nil {
		t.Fatal(err)
	}
	if content != "hello log" {
		t.Errorf("expected %q, got %q", "hello log", content)
	}
}

func TestReadLog_NoFile_ReturnsEmpty(t *testing.T) {
	tempHome(t)

	content, err := ReadLog()
	if err != nil {
		t.Errorf("expected no error for missing file, got: %v", err)
	}
	if content != "" {
		t.Errorf("expected empty string, got %q", content)
	}
}

// --- LogSize ---

func TestLogSize_ReturnsFileSize(t *testing.T) {
	_, logPath := tempHome(t)
	data := []byte("size test content")
	writeLog(t, logPath, data)

	if got := LogSize(); got != int64(len(data)) {
		t.Errorf("expected %d, got %d", len(data), got)
	}
}

func TestLogSize_NoFile_ReturnsZero(t *testing.T) {
	tempHome(t)

	if got := LogSize(); got != 0 {
		t.Errorf("expected 0 for missing file, got %d", got)
	}
}

// --- SanitizePath ---

func TestSanitizePath_HomePrefix_Replaced(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	p := filepath.Join(home, "some", "path")
	got := SanitizePath(p)
	want := "~/some/path"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestSanitizePath_NoHomePrefix_Unchanged(t *testing.T) {
	p := "/tmp/unrelated/path"
	if got := SanitizePath(p); got != p {
		t.Errorf("expected path unchanged, got %q", got)
	}
}

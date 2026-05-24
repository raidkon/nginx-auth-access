package verifylog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWriteClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "logs", "verify.log")
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	l.Write(Entry{Status: 204, Reason: "session_ok", Login: "alice"})
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"reason":"session_ok"`) {
		t.Fatalf("log content: %s", b)
	}
}

func TestNewEmptyPathReturnsNil(t *testing.T) {
	l, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	if l != nil {
		t.Fatal("expected nil logger")
	}
}

func TestWriteNilLoggerSafe(t *testing.T) {
	var l *Logger
	l.Write(Entry{Status: 401, Reason: "noop"})
}

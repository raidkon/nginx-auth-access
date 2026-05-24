// Package verifylog — одна строка JSON на событие auth_request (/internal/verify).
package verifylog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry — строка в логе (удобно для jq/rg).
type Entry struct {
	Time           string `json:"ts"`
	Host           string `json:"host,omitempty"`
	OriginalURI    string `json:"original_uri,omitempty"`
	OriginalMethod string `json:"original_method,omitempty"`
	ClientIP       string `json:"client_ip,omitempty"`
	RealClientIP   string `json:"real_client_ip"`
	RemoteAddr     string `json:"remote_addr,omitempty"`
	UserAgent      string `json:"user_agent,omitempty"`
	Status         int    `json:"status"`
	Reason         string `json:"reason"`
	Login          string `json:"login,omitempty"`
}

// Logger пишет в файл под мьютексом.
type Logger struct {
	path string
	mu   sync.Mutex
	f    *os.File
}

// New создаёт логгер; path — файл (каталог создаётся).
func New(path string) (*Logger, error) {
	if path == "" {
		return nil, nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &Logger{path: path, f: f}, nil
}

// Close закрывает файл.
func (l *Logger) Close() error {
	if l == nil || l.f == nil {
		return nil
	}
	return l.f.Close()
}

// Write добавляет запись (безопасно из нескольких горутин).
func (l *Logger) Write(e Entry) {
	if l == nil || l.f == nil {
		return
	}
	if e.Time == "" {
		e.Time = time.Now().UTC().Format(time.RFC3339Nano)
	}
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	l.mu.Lock()
	_, _ = l.f.Write(append(b, '\n'))
	l.mu.Unlock()
}

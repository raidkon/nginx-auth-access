package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFileAppliesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Listen.Public != ":80" || f.Listen.Verify != ":81" {
		t.Fatalf("listen defaults: %+v", f.Listen)
	}
	if f.VerifyLogPath != "/data/logs/verify.log" {
		t.Fatalf("verify_log_path: %q", f.VerifyLogPath)
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	initial := &File{
		SigningKey:    "test-signing-key-32-bytes!!!",
		PanelHostname: "access.example.com",
		CookieSecure:  true,
		Users: []User{{
			Login:          "alice",
			PasswordBcrypt: "$2a$10$abcdefghijklmnopqrstuv",
			TOTPSecret:     "JBSWY3DPEHPK3PXP",
			Admin:          true,
		}},
	}
	if err := Save(path, initial); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.PanelHostname != "access.example.com" {
		t.Fatalf("panel_hostname: %q", loaded.PanelHostname)
	}
	if len(loaded.Users) != 1 || loaded.Users[0].Login != "alice" {
		t.Fatalf("users: %+v", loaded.Users)
	}
}

func TestValidateRejectsShortSigningKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`signing_key = "short"`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "signing_key") {
		t.Fatalf("expected signing_key error, got %v", err)
	}
}

func TestExpandEnv(t *testing.T) {
	t.Setenv("TEST_PANEL_HOST", "panel.test")
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `signing_key = "test-signing-key-32-bytes!!!"
panel_hostname = "${TEST_PANEL_HOST}"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.PanelHostname != "panel.test" {
		t.Fatalf("panel_hostname: %q", f.PanelHostname)
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "secret") {
		t.Fatal("expected password match")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("expected password mismatch")
	}
}

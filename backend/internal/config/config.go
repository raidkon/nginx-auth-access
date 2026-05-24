package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/crypto/bcrypt"
)

// User — запись [[users]] в config.toml (пароль — только bcrypt-хэш).
type User struct {
	Login          string `toml:"login"`
	PasswordBcrypt string `toml:"password_bcrypt"`
	TOTPSecret     string `toml:"totp_secret"`
	Admin          bool   `toml:"admin"`
}

// Listen — адреса HTTP-серверов.
type Listen struct {
	Public string `toml:"public"`
	Verify string `toml:"verify"`
}

// File — полный config.toml (настройки сервиса и пользователи; по умолчанию /etc/nginx-auth-access/config.toml).
type File struct {
	SigningKey    string `toml:"signing_key"`
	PanelHostname string `toml:"panel_hostname"`
	CookieDomain  string `toml:"cookie_domain"`
	CookieSecure  bool   `toml:"cookie_secure"`
	NetSafeAccess string `toml:"net_safe_access"`
	VerifyLogPath string `toml:"verify_log_path"`
	StaticDir     string `toml:"static_dir"`
	Listen        Listen `toml:"listen"`
	Users         []User `toml:"users"`
}

var saveMu sync.Mutex

func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			f := &File{}
			f.ApplyDefaults()
			return f, nil
		}
		return nil, err
	}
	var f File
	if err := toml.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	f.ApplyDefaults()
	f.ExpandEnv()
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return &f, nil
}

func (f *File) ApplyDefaults() {
	if strings.TrimSpace(f.VerifyLogPath) == "" {
		f.VerifyLogPath = "/data/logs/verify.log"
	}
	if strings.TrimSpace(f.Listen.Public) == "" {
		f.Listen.Public = ":80"
	}
	if strings.TrimSpace(f.Listen.Verify) == "" {
		f.Listen.Verify = ":81"
	}
	if strings.TrimSpace(f.SigningKey) == "" {
		f.SigningKey = "dev-insecure-key-change-me-32b"
	}
}

func (f *File) ExpandEnv() {
	f.SigningKey = os.ExpandEnv(f.SigningKey)
	f.PanelHostname = os.ExpandEnv(f.PanelHostname)
	f.CookieDomain = os.ExpandEnv(f.CookieDomain)
	f.NetSafeAccess = os.ExpandEnv(f.NetSafeAccess)
	f.VerifyLogPath = os.ExpandEnv(f.VerifyLogPath)
	f.StaticDir = os.ExpandEnv(f.StaticDir)
	f.Listen.Public = os.ExpandEnv(f.Listen.Public)
	f.Listen.Verify = os.ExpandEnv(f.Listen.Verify)
}

func (f *File) Validate() error {
	if len(strings.TrimSpace(f.SigningKey)) < 16 {
		return fmt.Errorf("signing_key: нужно не менее 16 символов")
	}
	return nil
}

func Save(path string, f *File) error {
	saveMu.Lock()
	defer saveMu.Unlock()
	b, err := toml.Marshal(f)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

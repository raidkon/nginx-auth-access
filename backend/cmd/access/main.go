package main

import (
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/raidkon/nginx-auth-access/backend/internal/api"
	"github.com/raidkon/nginx-auth-access/backend/internal/config"
	"github.com/raidkon/nginx-auth-access/backend/internal/netutil"
	"github.com/raidkon/nginx-auth-access/backend/internal/uiembed"
	"github.com/raidkon/nginx-auth-access/backend/internal/verifylog"
)

func main() {
	configFlag := flag.String("config", "/etc/nginx-auth-access/config.toml", "путь к TOML-конфигу")
	flag.Parse()
	cfgPath := *configFlag
	if v := os.Getenv("ACCESS_CONFIG_PATH"); v != "" {
		cfgPath = v
	}
	file, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("access: config %s: %v", cfgPath, err)
	}

	safe := netutil.ParseList(file.NetSafeAccess)

	vlogPath := strings.TrimSpace(file.VerifyLogPath)
	var vlog *verifylog.Logger
	if vlogPath != "-" && vlogPath != "off" {
		vlog, err = verifylog.New(vlogPath)
		if err != nil {
			log.Printf("access: лог auth_request (%s): %v — продолжаю без файла", vlogPath, err)
		} else {
			log.Printf("access: auth_request → %s", vlogPath)
		}
	}

	h := &api.Handler{
		ConfigPath:      cfgPath,
		SigningKey:      []byte(file.SigningKey),
		CookieDomain:    file.CookieDomain,
		CookieSecure:    file.CookieSecure,
		SafeAccessCIDRs: safe,
		VerifyLog:       vlog,
	}

	staticFS, staticLabel := staticFileSystem(file.StaticDir)
	if _, err := fs.Stat(staticFS, "index.html"); err != nil {
		log.Fatalf("access: в статике (%s) нет index.html: %v", staticLabel, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/login", h.Login)
	mux.HandleFunc("/api/v1/session", h.Session)
	mux.HandleFunc("/api/v1/logout", h.Logout)
	mux.HandleFunc("/api/v1/users/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			h.DeleteUser(w, r)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListUsers(w, r)
		case http.MethodPost:
			h.AddUser(w, r)
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})

	mux.Handle("/", spaHandler{label: staticLabel, fsys: staticFS})

	verify := http.NewServeMux()
	verify.HandleFunc("/internal/verify", h.HandleVerify)

	go func() {
		addr := file.Listen.Verify
		s := &http.Server{Addr: addr, Handler: logReq(verify), ReadHeaderTimeout: 10 * time.Second}
		log.Printf("access verify listening %s", addr)
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	addr := file.Listen.Public
	s := &http.Server{Addr: addr, Handler: logReq(mux), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("access public listening %s (static %s)", addr, staticLabel)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func staticFileSystem(staticDir string) (fs.FS, string) {
	if d := strings.TrimSpace(staticDir); d != "" {
		dir := os.DirFS(d)
		if _, statErr := fs.Stat(dir, "index.html"); statErr == nil {
			return dir, d
		} else {
			log.Printf("access: static_dir=%q: index.html недоступен (%v), используем встроенный SPA", d, statErr)
		}
	}
	root, err := uiembed.SPARoot()
	if err != nil {
		log.Fatalf("access: встроенный SPA: %v", err)
	}
	return root, "(embedded)"
}

func logReq(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

type spaHandler struct {
	label string
	fsys  fs.FS
}

func (s spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	rel := strings.TrimPrefix(r.URL.Path, "/")
	if rel == "" {
		rel = "index.html"
	}
	if strings.Contains(rel, "..") {
		http.NotFound(w, r)
		return
	}
	b, err := fs.ReadFile(s.fsys, rel)
	if err != nil {
		b, err = fs.ReadFile(s.fsys, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		rel = "index.html"
	}
	w.Header().Set("Content-Type", contentType("/"+rel))
	_, _ = w.Write(b)
}

func contentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".js"):
		return "text/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".mjs"):
		return "text/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".webp"):
		return "image/webp"
	case strings.HasSuffix(path, ".woff2"):
		return "font/woff2"
	case strings.HasSuffix(path, ".woff"):
		return "font/woff"
	case strings.HasSuffix(path, ".ttf"):
		return "font/ttf"
	case strings.HasSuffix(path, ".json"):
		return "application/json; charset=utf-8"
	case strings.HasSuffix(path, ".map"):
		return "application/json; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

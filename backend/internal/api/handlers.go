package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/raidkon/nginx-auth-access/backend/internal/config"
	"github.com/raidkon/nginx-auth-access/backend/internal/netutil"
	"github.com/raidkon/nginx-auth-access/backend/internal/session"
	"github.com/raidkon/nginx-auth-access/backend/internal/totp"
	"github.com/raidkon/nginx-auth-access/backend/internal/verifylog"
)

const CookieName = "ACCESS_ZONE"

type Handler struct {
	ConfigPath      string
	SigningKey      []byte
	CookieDomain    string
	CookieSecure    bool
	SafeAccessCIDRs []string
	VerifyLog       *verifylog.Logger
}

type loginBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	TOTP     string `json:"totp"`
	Period   string `json:"period"` // "30m", "1h", "3h", "8h", "24h"
}

func (h *Handler) parsePeriod(s string) time.Duration {
	switch strings.TrimSpace(s) {
	case "30m":
		return 30 * time.Minute
	case "1h":
		return time.Hour
	case "3h":
		return 3 * time.Hour
	case "8h":
		return 8 * time.Hour
	case "24h":
		return 24 * time.Hour
	default:
		return 30 * time.Minute
	}
}

func (h *Handler) json(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) clientIP(r *http.Request) string {
	return session.ClientIP(
		r.RemoteAddr,
		r.Header.Get("X-Real-IP"),
		r.Header.Get("X-Forwarded-For"),
	)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.json(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		return
	}
	var body loginBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.json(w, http.StatusBadRequest, map[string]string{"error": "json"})
		return
	}
	cfg, err := config.Load(h.ConfigPath)
	if err != nil {
		h.json(w, http.StatusInternalServerError, map[string]string{"error": "config"})
		return
	}
	ip := h.clientIP(r)
	ttl := h.parsePeriod(body.Period)

	var loginOK bool
	var subj string
	if len(cfg.Users) == 0 {
		if body.Login == "noauth" && body.Password == "noauth" && totp.Valid("", body.TOTP) {
			loginOK = true
			subj = "noauth"
		}
	} else {
		for _, u := range cfg.Users {
			if u.Login != body.Login {
				continue
			}
			if !config.CheckPassword(u.PasswordBcrypt, body.Password) {
				break
			}
			if !totp.Valid(u.TOTPSecret, body.TOTP) {
				break
			}
			loginOK = true
			subj = u.Login
			break
		}
	}
	if !loginOK {
		h.json(w, http.StatusUnauthorized, map[string]string{"error": "credentials"})
		return
	}
	token, err := session.Sign(h.SigningKey, ip, subj, ttl)
	if err != nil {
		h.json(w, http.StatusInternalServerError, map[string]string{"error": "session"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.CookieSecure,
		Domain:   h.CookieDomain,
	})
	h.json(w, http.StatusOK, map[string]any{"ok": true, "login": subj, "bootstrap": len(cfg.Users) == 0})
}

func (h *Handler) Session(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.json(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		return
	}
	ck, err := r.Cookie(CookieName)
	if err != nil {
		h.json(w, http.StatusUnauthorized, map[string]any{"ok": false})
		return
	}
	ip := h.clientIP(r)
	cl, err := session.Parse(h.SigningKey, ck.Value)
	if err != nil || cl.IP != ip {
		h.json(w, http.StatusUnauthorized, map[string]any{"ok": false})
		return
	}
	cfg, _ := config.Load(h.ConfigPath)
	admin := len(cfg.Users) == 0 && cl.Login == "noauth"
	if !admin {
		for _, u := range cfg.Users {
			if u.Login == cl.Login && u.Admin {
				admin = true
				break
			}
		}
	}
	h.json(w, http.StatusOK, map[string]any{"ok": true, "login": cl.Login, "admin": admin, "bootstrap": len(cfg.Users) == 0})
}

func (h *Handler) requireSession(r *http.Request) (*session.Claims, error) {
	ck, err := r.Cookie(CookieName)
	if err != nil {
		return nil, err
	}
	ip := h.clientIP(r)
	cl, err := session.Parse(h.SigningKey, ck.Value)
	if err != nil || cl.IP != ip {
		return nil, errors.New("bad session")
	}
	return cl, nil
}

func (h *Handler) requireAdmin(r *http.Request) (*session.Claims, error) {
	cl, err := h.requireSession(r)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(h.ConfigPath)
	if err != nil {
		return nil, err
	}
	if len(cfg.Users) == 0 && cl.Login == "noauth" {
		return cl, nil
	}
	for _, u := range cfg.Users {
		if u.Login == cl.Login && u.Admin {
			return cl, nil
		}
	}
	return nil, errors.New("forbidden")
}

type userDTO struct {
	Login      string `json:"login"`
	Password   string `json:"password"`
	TOTPSecret string `json:"totp_secret"`
	Admin      bool   `json:"admin"`
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.json(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		return
	}
	if _, err := h.requireAdmin(r); err != nil {
		h.json(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	cfg, err := config.Load(h.ConfigPath)
	if err != nil {
		h.json(w, http.StatusInternalServerError, map[string]string{"error": "config"})
		return
	}
	out := make([]map[string]any, 0, len(cfg.Users))
	for _, u := range cfg.Users {
		hasTotp := strings.TrimSpace(u.TOTPSecret) != ""
		out = append(out, map[string]any{"login": u.Login, "admin": u.Admin, "has_totp": hasTotp})
	}
	h.json(w, http.StatusOK, map[string]any{"users": out})
}

func (h *Handler) AddUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.json(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		return
	}
	if _, err := h.requireAdmin(r); err != nil {
		h.json(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var body userDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.json(w, http.StatusBadRequest, map[string]string{"error": "json"})
		return
	}
	body.Login = strings.TrimSpace(body.Login)
	if body.Login == "" || body.Password == "" {
		h.json(w, http.StatusBadRequest, map[string]string{"error": "fields"})
		return
	}
	hash, err := config.HashPassword(body.Password)
	if err != nil {
		h.json(w, http.StatusInternalServerError, map[string]string{"error": "hash"})
		return
	}
	cfg, err := config.Load(h.ConfigPath)
	if err != nil {
		h.json(w, http.StatusInternalServerError, map[string]string{"error": "config"})
		return
	}
	for _, u := range cfg.Users {
		if u.Login == body.Login {
			h.json(w, http.StatusConflict, map[string]string{"error": "exists"})
			return
		}
	}
	// первый пользователь в системе — всегда admin
	if len(cfg.Users) == 0 {
		body.Admin = true
	}
	cfg.Users = append(cfg.Users, config.User{
		Login:          body.Login,
		PasswordBcrypt: hash,
		TOTPSecret:     strings.TrimSpace(body.TOTPSecret),
		Admin:          body.Admin,
	})
	if err := config.Save(h.ConfigPath, cfg); err != nil {
		h.json(w, http.StatusInternalServerError, map[string]string{"error": "save"})
		return
	}
	h.json(w, http.StatusCreated, map[string]any{"ok": true})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.json(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		return
	}
	if _, err := h.requireAdmin(r); err != nil {
		h.json(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	login := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	login = strings.TrimSpace(login)
	if login == "" {
		h.json(w, http.StatusBadRequest, map[string]string{"error": "login"})
		return
	}
	cfg, err := config.Load(h.ConfigPath)
	if err != nil {
		h.json(w, http.StatusInternalServerError, map[string]string{"error": "config"})
		return
	}
	next := cfg.Users[:0]
	for _, u := range cfg.Users {
		if u.Login != login {
			next = append(next, u)
		}
	}
	if len(next) == len(cfg.Users) {
		h.json(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	cfg.Users = next
	if err := config.Save(h.ConfigPath, cfg); err != nil {
		h.json(w, http.StatusInternalServerError, map[string]string{"error": "save"})
		return
	}
	h.json(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.json(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.CookieSecure,
		Domain:   h.CookieDomain,
	})
	h.json(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) authRequestEntry(r *http.Request) verifylog.Entry {
	ip := h.clientIP(r)
	host := strings.TrimSpace(r.Host)
	if xfh := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); xfh != "" {
		host = xfh
	}
	origMethod := r.Header.Get("X-Original-Method")
	if origMethod == "" {
		origMethod = r.Method
	}
	return verifylog.Entry{
		Host:           host,
		OriginalURI:    r.Header.Get("X-Original-URI"),
		OriginalMethod: origMethod,
		ClientIP:       ip,
		RealClientIP:   session.FirstForwardedIP(r.Header.Get("X-Access-Client-IP")),
		RemoteAddr:     r.RemoteAddr,
		UserAgent:      r.Header.Get("X-Original-User-Agent"),
	}
}

// HandleVerify — только :81, сабреквесты nginx auth_request.
func (h *Handler) HandleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		e := h.authRequestEntry(r)
		e.Status = http.StatusMethodNotAllowed
		e.Reason = "method_not_allowed"
		h.VerifyLog.Write(e)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	h.verify(w, r)
}

func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	entry := h.authRequestEntry(r)

	if r.URL.Path != "/internal/verify" {
		entry.Status = http.StatusUnauthorized
		entry.Reason = "wrong_path"
		w.WriteHeader(http.StatusUnauthorized)
		h.VerifyLog.Write(entry)
		return
	}
	ip := entry.ClientIP
	if len(h.SafeAccessCIDRs) > 0 && netutil.InAnyCIDR(ip, h.SafeAccessCIDRs) {
		entry.Status = http.StatusNoContent
		entry.Reason = "safe_cidr"
		w.WriteHeader(http.StatusNoContent)
		h.VerifyLog.Write(entry)
		return
	}
	ck, err := r.Cookie(CookieName)
	if err != nil {
		entry.Status = http.StatusUnauthorized
		entry.Reason = "no_cookie"
		w.WriteHeader(http.StatusUnauthorized)
		h.VerifyLog.Write(entry)
		return
	}
	cl, err := session.Parse(h.SigningKey, ck.Value)
	if err != nil {
		entry.Status = http.StatusUnauthorized
		entry.Reason = "invalid_token"
		w.WriteHeader(http.StatusUnauthorized)
		h.VerifyLog.Write(entry)
		return
	}
	if cl.IP != ip {
		entry.Status = http.StatusUnauthorized
		entry.Reason = "ip_mismatch"
		entry.Login = cl.Login
		w.WriteHeader(http.StatusUnauthorized)
		h.VerifyLog.Write(entry)
		return
	}
	entry.Status = http.StatusNoContent
	entry.Reason = "session_ok"
	entry.Login = cl.Login
	w.WriteHeader(http.StatusNoContent)
	h.VerifyLog.Write(entry)
}

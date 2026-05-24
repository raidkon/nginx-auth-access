package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/raidkon/nginx-auth-access/backend/internal/config"
	"github.com/raidkon/nginx-auth-access/backend/internal/session"
	"github.com/raidkon/nginx-auth-access/backend/internal/verifylog"
)

const testSigningKey = "test-signing-key-32-bytes!!!"

func setupHandler(t *testing.T, users []config.User) (*Handler, string) {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	cfg := &config.File{
		SigningKey: testSigningKey,
		Users:      users,
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	vlog, err := verifylog.New(filepath.Join(dir, "verify.log"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = vlog.Close() })

	return &Handler{
		ConfigPath:      cfgPath,
		SigningKey:      []byte(testSigningKey),
		CookieSecure:    true,
		CookieDomain:    ".example.com",
		SafeAccessCIDRs: []string{"192.168.0.0/24"},
		VerifyLog:       vlog,
	}, dir
}

func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(dst); err != nil {
		t.Fatalf("decode json: %v body=%q", err, rr.Body.String())
	}
}

func sessionRequest(t *testing.T, h *Handler, method, target, ip, login string, body any) *http.Request {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	req.RemoteAddr = ip + ":12345"
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if login != "" {
		token, err := session.Sign(h.SigningKey, strings.Split(ip, "/")[0], login, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	}
	return req
}

func TestParsePeriod(t *testing.T) {
	h := &Handler{}
	if got := h.parsePeriod("8h"); got != 8*time.Hour {
		t.Fatalf("parsePeriod: %v", got)
	}
	if got := h.parsePeriod("unknown"); got != 30*time.Minute {
		t.Fatalf("default period: %v", got)
	}
}

func TestLoginBootstrap(t *testing.T) {
	h, _ := setupHandler(t, nil)
	body := map[string]string{
		"login": "noauth", "password": "noauth", "totp": "000000", "period": "1h",
	}
	req := sessionRequest(t, h, http.MethodPost, "/api/v1/login", "203.0.113.1", "", body)
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
	ck := rr.Result().Cookies()[0]
	if ck.Name != CookieName || !ck.Secure || !strings.HasSuffix(ck.Domain, "example.com") {
		t.Fatalf("cookie: %+v", ck)
	}
	var resp map[string]any
	decodeJSON(t, rr, &resp)
	if resp["bootstrap"] != true {
		t.Fatalf("bootstrap response: %+v", resp)
	}
}

func TestLoginWrongMethod(t *testing.T) {
	h, _ := setupHandler(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/login", nil)
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestLoginWithUser(t *testing.T) {
	hash, err := config.HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	secret := "JBSWY3DPEHPK3PXP"
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	h, _ := setupHandler(t, []config.User{{
		Login: "alice", PasswordBcrypt: hash, TOTPSecret: secret, Admin: true,
	}})
	body := map[string]string{
		"login": "alice", "password": "secret", "totp": code, "period": "30m",
	}
	req := sessionRequest(t, h, http.MethodPost, "/api/v1/login", "203.0.113.2", "", body)
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestLoginBadCredentials(t *testing.T) {
	hash, err := config.HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	h, _ := setupHandler(t, []config.User{{
		Login: "alice", PasswordBcrypt: hash, TOTPSecret: "JBSWY3DPEHPK3PXP", Admin: true,
	}})
	body := map[string]string{
		"login": "alice", "password": "wrong", "totp": "000000", "period": "30m",
	}
	req := sessionRequest(t, h, http.MethodPost, "/api/v1/login", "203.0.113.3", "", body)
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestSessionAndLogout(t *testing.T) {
	h, _ := setupHandler(t, nil)

	loginReq := sessionRequest(t, h, http.MethodPost, "/api/v1/login", "203.0.113.4", "", map[string]string{
		"login": "noauth", "password": "noauth", "totp": "000000", "period": "1h",
	})
	loginRR := httptest.NewRecorder()
	h.Login(loginRR, loginReq)
	ck := loginRR.Result().Cookies()[0]

	sessReq := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	sessReq.RemoteAddr = "203.0.113.4:12345"
	sessReq.AddCookie(ck)
	sessRR := httptest.NewRecorder()
	h.Session(sessRR, sessReq)
	if sessRR.Code != http.StatusOK {
		t.Fatalf("session status %d", sessRR.Code)
	}
	var sess map[string]any
	decodeJSON(t, sessRR, &sess)
	if sess["admin"] != true {
		t.Fatalf("expected bootstrap admin: %+v", sess)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
	logoutRR := httptest.NewRecorder()
	h.Logout(logoutRR, logoutReq)
	if logoutRR.Code != http.StatusOK {
		t.Fatalf("logout status %d", logoutRR.Code)
	}
	clearCookie := logoutRR.Result().Cookies()[0]
	if clearCookie.MaxAge != -1 {
		t.Fatalf("logout cookie: %+v", clearCookie)
	}
}

func TestUsersCRUD(t *testing.T) {
	h, _ := setupHandler(t, nil)
	ip := "203.0.113.5"
	secret := "JBSWY3DPEHPK3PXP"

	loginRR := httptest.NewRecorder()
	h.Login(loginRR, sessionRequest(t, h, http.MethodPost, "/api/v1/login", ip, "", map[string]string{
		"login": "noauth", "password": "noauth", "totp": "000000", "period": "1h",
	}))
	adminCookie := loginRR.Result().Cookies()[0]

	listRR := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	listReq.RemoteAddr = ip + ":12345"
	listReq.AddCookie(adminCookie)
	h.ListUsers(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list status %d", listRR.Code)
	}

	addRR := httptest.NewRecorder()
	h.AddUser(addRR, sessionRequest(t, h, http.MethodPost, "/api/v1/users", ip, "noauth", map[string]any{
		"login": "bob", "password": "pass", "totp_secret": secret, "admin": false,
	}))
	if addRR.Code != http.StatusCreated {
		t.Fatalf("add status %d body=%s", addRR.Code, addRR.Body.String())
	}

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	bobLoginRR := httptest.NewRecorder()
	h.Login(bobLoginRR, sessionRequest(t, h, http.MethodPost, "/api/v1/login", ip, "", map[string]string{
		"login": "bob", "password": "pass", "totp": code, "period": "1h",
	}))
	if bobLoginRR.Code != http.StatusOK {
		t.Fatalf("bob login status %d body=%s", bobLoginRR.Code, bobLoginRR.Body.String())
	}
	bobCookie := bobLoginRR.Result().Cookies()[0]

	listRR2 := httptest.NewRecorder()
	listReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	listReq2.RemoteAddr = ip + ":12345"
	listReq2.AddCookie(bobCookie)
	h.ListUsers(listRR2, listReq2)
	if listRR2.Code != http.StatusOK {
		t.Fatalf("list after add status %d body=%s", listRR2.Code, listRR2.Body.String())
	}
	var listed struct {
		Users []map[string]any `json:"users"`
	}
	decodeJSON(t, listRR2, &listed)
	if len(listed.Users) != 1 || listed.Users[0]["login"] != "bob" {
		t.Fatalf("users after add: %+v", listed.Users)
	}

	delRR := httptest.NewRecorder()
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/users/bob", nil)
	delReq.RemoteAddr = ip + ":12345"
	delReq.AddCookie(bobCookie)
	h.DeleteUser(delRR, delReq)
	if delRR.Code != http.StatusOK {
		t.Fatalf("delete status %d", delRR.Code)
	}
}

func TestVerifySafeCIDRAndSession(t *testing.T) {
	h, _ := setupHandler(t, nil)

	safeReq := httptest.NewRequest(http.MethodGet, "/internal/verify", nil)
	safeReq.RemoteAddr = "192.168.0.10:12345"
	safeReq.Header.Set("X-Original-URI", "/secret")
	safeRR := httptest.NewRecorder()
	h.HandleVerify(safeRR, safeReq)
	if safeRR.Code != http.StatusNoContent {
		t.Fatalf("safe cidr status %d", safeRR.Code)
	}

	noCookieRR := httptest.NewRecorder()
	noCookieReq := httptest.NewRequest(http.MethodGet, "/internal/verify", nil)
	noCookieReq.RemoteAddr = "8.8.8.8:12345"
	h.HandleVerify(noCookieRR, noCookieReq)
	if noCookieRR.Code != http.StatusUnauthorized {
		t.Fatalf("no cookie status %d", noCookieRR.Code)
	}

	token, err := session.Sign(h.SigningKey, "8.8.8.8", "noauth", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	okRR := httptest.NewRecorder()
	okReq := httptest.NewRequest(http.MethodGet, "/internal/verify", nil)
	okReq.RemoteAddr = "8.8.8.8:12345"
	okReq.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	h.HandleVerify(okRR, okReq)
	if okRR.Code != http.StatusNoContent {
		t.Fatalf("session ok status %d", okRR.Code)
	}

	mismatchRR := httptest.NewRecorder()
	mismatchReq := httptest.NewRequest(http.MethodGet, "/internal/verify", nil)
	mismatchReq.RemoteAddr = "8.8.4.4:12345"
	mismatchReq.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	h.HandleVerify(mismatchRR, mismatchReq)
	if mismatchRR.Code != http.StatusUnauthorized {
		t.Fatalf("ip mismatch status %d", mismatchRR.Code)
	}
}

func TestVerifyWrongMethod(t *testing.T) {
	h, _ := setupHandler(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/internal/verify", nil)
	rr := httptest.NewRecorder()
	h.HandleVerify(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestRequireAdminForbidden(t *testing.T) {
	hash, _ := config.HashPassword("secret")
	h, _ := setupHandler(t, []config.User{{
		Login: "bob", PasswordBcrypt: hash, TOTPSecret: "JBSWY3DPEHPK3PXP", Admin: false,
	}})
	req := sessionRequest(t, h, http.MethodGet, "/api/v1/users", "203.0.113.6", "bob", nil)
	rr := httptest.NewRecorder()
	h.ListUsers(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestSessionWithoutCookie(t *testing.T) {
	h, _ := setupHandler(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	rr := httptest.NewRecorder()
	h.Session(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestAddUserDuplicate(t *testing.T) {
	hash, _ := config.HashPassword("secret")
	h, _ := setupHandler(t, []config.User{{
		Login: "alice", PasswordBcrypt: hash, TOTPSecret: "JBSWY3DPEHPK3PXP", Admin: true,
	}})
	req := sessionRequest(t, h, http.MethodPost, "/api/v1/users", "203.0.113.7", "alice", map[string]any{
		"login": "alice", "password": "x", "totp_secret": "JBSWY3DPEHPK3PXP",
	})
	rr := httptest.NewRecorder()
	h.AddUser(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestDeleteUserNotFound(t *testing.T) {
	hash, _ := config.HashPassword("secret")
	h, _ := setupHandler(t, []config.User{{
		Login: "alice", PasswordBcrypt: hash, TOTPSecret: "JBSWY3DPEHPK3PXP", Admin: true,
	}})
	req := sessionRequest(t, h, http.MethodDelete, "/api/v1/users/missing", "203.0.113.8", "alice", nil)
	rr := httptest.NewRecorder()
	h.DeleteUser(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestHandleVerifyHead(t *testing.T) {
	h, _ := setupHandler(t, nil)
	token, err := session.Sign(h.SigningKey, "8.8.8.8", "noauth", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodHead, "/internal/verify", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	rr := httptest.NewRecorder()
	h.HandleVerify(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestAuthRequestEntryForwardedHost(t *testing.T) {
	h, _ := setupHandler(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/internal/verify", nil)
	req.Host = "access.example.com"
	req.Header.Set("X-Forwarded-Host", "public.example.com")
	req.Header.Set("X-Original-URI", "/app")
	req.Header.Set("X-Access-Client-IP", "198.51.100.1")
	entry := h.authRequestEntry(req)
	if entry.Host != "public.example.com" || entry.OriginalURI != "/app" {
		t.Fatalf("entry: %+v", entry)
	}
	if entry.RealClientIP != "198.51.100.1" {
		t.Fatalf("real client ip: %q", entry.RealClientIP)
	}
}

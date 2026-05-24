package session

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	IP    string `json:"ip"`
	Login string `json:"login"`
	jwt.RegisteredClaims
}

func Sign(secret []byte, ip, login string, ttl time.Duration) (string, error) {
	if len(secret) < 16 {
		return "", errors.New("signing key too short")
	}
	now := time.Now()
	c := Claims{
		IP:    ip,
		Login: login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, &c)
	return t.SignedString(secret)
}

func Parse(secret []byte, tokenString string) (*Claims, error) {
	c := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, c, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return c, nil
}

func ClientIP(remoteAddr, xRealIP, xForwardedFor string) string {
	if xRealIP != "" {
		return strings.TrimSpace(xRealIP)
	}
	if xForwardedFor != "" {
		s := strings.TrimSpace(xForwardedFor)
		if i := strings.IndexByte(s, ','); i >= 0 {
			s = strings.TrimSpace(s[:i])
		}
		if host, _, err := net.SplitHostPort(s); err == nil {
			return host
		}
		if ip := net.ParseIP(s); ip != nil {
			return ip.String()
		}
		return s
	}
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

// FirstForwardedIP — первая запись из заголовка в стиле X-Forwarded-For / X-Access-Client-IP.
func FirstForwardedIP(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, ','); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if h, _, err := net.SplitHostPort(s); err == nil {
		return h
	}
	if ip := net.ParseIP(s); ip != nil {
		return ip.String()
	}
	return s
}

// EffectiveClientIP как ClientIP, но если соединение с nginx после stream даёт loopback (127.0.0.1),
// и приходит пара заголовков X-Access-Real-IP-Secret / X-Access-Client-IP с верным sharedSecret,
// подставляется адрес из X-Access-Client-IP (nginx кладёт туда CF-Connecting-IP или X-Forwarded-For).
func EffectiveClientIP(remoteAddr, xRealIP, xForwardedFor, sharedSecret, headerSecret, headerClientIP string) string {
	base := ClientIP(remoteAddr, xRealIP, xForwardedFor)
	if sharedSecret == "" || headerSecret == "" || headerClientIP == "" {
		return base
	}
	if len(sharedSecret) != len(headerSecret) || subtle.ConstantTimeCompare([]byte(sharedSecret), []byte(headerSecret)) != 1 {
		return base
	}
	baseHost := strings.TrimSpace(base)
	if h, _, err := net.SplitHostPort(baseHost); err == nil {
		baseHost = h
	}
	baseIP := net.ParseIP(baseHost)
	if baseIP == nil || !baseIP.IsLoopback() {
		return base
	}
	alt := strings.TrimSpace(headerClientIP)
	if i := strings.IndexByte(alt, ','); i >= 0 {
		alt = strings.TrimSpace(alt[:i])
	}
	if h, _, err := net.SplitHostPort(alt); err == nil {
		alt = h
	}
	altIP := net.ParseIP(alt)
	if altIP == nil {
		return base
	}
	return altIP.String()
}

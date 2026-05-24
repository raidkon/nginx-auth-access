package netutil

import (
	"net"
	"strings"
)

// InAnyCIDR возвращает true, если addr входит хотя бы в одну сеть из списка (формат как net_safe_access в config.toml).
func InAnyCIDR(addr string, cidrs []string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	for _, raw := range cidrs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		_, n, err := net.ParseCIDR(raw)
		if err != nil {
			// одиночный IP
			single := net.ParseIP(raw)
			if single != nil && single.Equal(ip) {
				return true
			}
			continue
		}
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func ParseList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

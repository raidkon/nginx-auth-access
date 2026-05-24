package session

import (
	"testing"
	"time"
)

func TestSignParseRoundTrip(t *testing.T) {
	secret := []byte("test-signing-key-32-bytes!!!")
	token, err := Sign(secret, "192.168.1.10", "alice", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	cl, err := Parse(secret, token)
	if err != nil {
		t.Fatal(err)
	}
	if cl.IP != "192.168.1.10" || cl.Login != "alice" {
		t.Fatalf("claims: %+v", cl)
	}
}

func TestSignRejectsShortKey(t *testing.T) {
	if _, err := Sign([]byte("short"), "1.1.1.1", "u", time.Minute); err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name   string
		remote string
		real   string
		xff    string
		want   string
	}{
		{"x-real-ip", "127.0.0.1:1234", "203.0.113.1", "", "203.0.113.1"},
		{"xff first", "127.0.0.1:1234", "", "198.51.100.2, 10.0.0.1", "198.51.100.2"},
		{"remote addr", "192.168.0.5:8080", "", "", "192.168.0.5"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClientIP(tc.remote, tc.real, tc.xff); got != tc.want {
				t.Fatalf("ClientIP() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFirstForwardedIP(t *testing.T) {
	if got := FirstForwardedIP("1.2.3.4, 5.6.7.8"); got != "1.2.3.4" {
		t.Fatalf("FirstForwardedIP: %q", got)
	}
	if FirstForwardedIP("") != "" {
		t.Fatal("expected empty")
	}
}

func TestEffectiveClientIPWithTrustedHeaders(t *testing.T) {
	got := EffectiveClientIP(
		"127.0.0.1:1234",
		"",
		"",
		"shared-secret-1234567890",
		"shared-secret-1234567890",
		"198.51.100.9",
	)
	if got != "198.51.100.9" {
		t.Fatalf("EffectiveClientIP: %q", got)
	}
	got = EffectiveClientIP(
		"127.0.0.1:1234",
		"",
		"",
		"shared-secret-1234567890",
		"wrong",
		"198.51.100.9",
	)
	if got != "127.0.0.1" {
		t.Fatalf("wrong secret should keep loopback base: %q", got)
	}
}

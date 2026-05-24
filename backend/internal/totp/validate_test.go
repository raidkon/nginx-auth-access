package totp

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestValidBootstrapCode(t *testing.T) {
	if !Valid("", "000000") {
		t.Fatal("expected bootstrap code")
	}
	if Valid("", "123456") {
		t.Fatal("expected only 000000 for empty secret")
	}
}

func TestValidWithSecret(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !Valid(secret, code) {
		t.Fatal("expected valid TOTP")
	}
	if Valid(secret, "000000") {
		t.Fatal("expected invalid TOTP")
	}
	if Valid("!!!invalid!!!", "123456") {
		t.Fatal("expected invalid secret")
	}
}

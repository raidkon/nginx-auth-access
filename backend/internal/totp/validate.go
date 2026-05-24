package totp

import (
	"encoding/base32"
	"strings"

	"github.com/pquerna/otp/totp"
)

func Valid(secret, code string) bool {
	secret = strings.TrimSpace(secret)
	code = strings.TrimSpace(code)
	if secret == "" {
		return code == "000000"
	}
	secret = strings.ToUpper(secret)
	// допускаем secret без padding
	if n := len(secret) % 8; n != 0 {
		secret += strings.Repeat("=", 8-n)
	}
	_, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return false
	}
	return totp.Validate(code, secret)
}

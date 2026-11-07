package hashx_test

import (
	"testing"

	"github.com/santekno/sdk/hashx"
)

func TestVerifyPassword_MalformedFields(t *testing.T) {
	for name, h := range map[string]string{
		"bad version":  "$argon2id$v=x$m=1,t=1,p=1$AAAA$AAAA",
		"bad params":   "$argon2id$v=19$m=x$AAAA$AAAA",
		"bad salt b64": "$argon2id$v=19$m=8,t=1,p=1$!!!$AAAA",
		"bad key b64":  "$argon2id$v=19$m=8,t=1,p=1$AAAA$!!!",
		"wrong algo":   "$argon2i$v=19$m=8,t=1,p=1$AAAA$AAAA",
	} {
		t.Run(name, func(t *testing.T) {
			if hashx.VerifyPassword("pw", h) {
				t.Error("malformed hash must not verify")
			}
		})
	}
}

func TestBcrypt_InvalidCost(t *testing.T) {
	if _, err := hashx.HashPasswordBcrypt("pw", 99); err == nil {
		t.Error("expected error for out-of-range cost")
	}
}

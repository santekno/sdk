package hashx

import "golang.org/x/crypto/bcrypt"

// BcryptDefaultCost is the default work factor for bcrypt.
// Reasonable choice for 2026 hardware.
const BcryptDefaultCost = 12

// HashPasswordBcrypt hashes password with bcrypt at the given cost.
// Use cost = BcryptDefaultCost for sensible defaults.
func HashPasswordBcrypt(password string, cost int) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}
	if cost == 0 {
		cost = BcryptDefaultCost
	}
	out, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// VerifyPasswordBcrypt reports whether password matches the bcrypt hash.
func VerifyPasswordBcrypt(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword hashes a plaintext password with configured cost.
func HashPassword(password string, cost int) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// ComparePassword verifies a password against its hashed value.
func ComparePassword(hashed, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
}

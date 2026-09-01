// Package auth handles password hashing and JWT issuing/verification.
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword turns a plaintext password into a bcrypt hash that is
// safe to store in the database. bcrypt automatically salts the hash.
func HashPassword(plain string) (string, error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashBytes), nil
}

// CheckPassword compares a plaintext password against a stored bcrypt
// hash. It returns true if they match.
func CheckPassword(plain, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	return err == nil
}

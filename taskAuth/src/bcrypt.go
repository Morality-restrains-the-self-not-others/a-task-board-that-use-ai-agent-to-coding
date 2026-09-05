package main

import (
	"golang.org/x/crypto/bcrypt"
)

// bcryptCost is the bcrypt work factor for password hashing. Production keeps
// 12 (~195ms/hash). It is a var rather than a const so the test binary can lower
// it via TestMain — 300+ tests hash passwords during setup and cost 12 would
// add ~30s of pure CPU to the suite. The cost is embedded in the stored hash,
// so verification (checkPasswordHash) is unaffected by the generation cost.
var bcryptCost = 12

// hashPassword uses bcrypt to hash a plaintext password.
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// checkPasswordHash compares a plaintext password against a bcrypt hash.
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

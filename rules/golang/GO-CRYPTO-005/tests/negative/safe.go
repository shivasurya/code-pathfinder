// GO-CRYPTO-005 negative test cases — NONE should be detected
package main

import "golang.org/x/crypto/bcrypt"

func safePasswordHash(password string) (string, error) {
	// SAFE: bcrypt with proper work factor
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func safePasswordVerify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

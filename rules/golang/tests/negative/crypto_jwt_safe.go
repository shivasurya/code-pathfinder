// Crypto and JWT negative test cases — NONE should be detected
package main

import (
	"crypto/aes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"io"
	"net/http"
	"os"

	jwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// SAFE: Strong hash algorithms

func safeHash() {
	// SHA256 — collision resistant, NIST recommended
	h := sha256.New()
	h.Write([]byte("data"))
	sum := h.Sum(nil)
	_ = sum
}

func safeHash512() {
	h := sha512.New()
	h.Write([]byte("data"))
	_ = h.Sum(nil)
}

// SAFE: HMAC with SHA256

func safeHMAC() {
	key := []byte("secret-key")
	mac := hmac.New(sha256.New, key) // SAFE: HMAC-SHA256
	mac.Write([]byte("message"))
}

// SAFE: AES encryption

func safeAES() {
	key := make([]byte, 32) // AES-256
	io.ReadFull(rand.Reader, key)
	_, _ = aes.NewCipher(key) // SAFE: AES
}

// SAFE: bcrypt for password hashing

func safePasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func safePasswordVerify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// SAFE: JWT with proper signing algorithm

func safeJWTSign() string {
	token := jwt.New(jwt.SigningMethodHS256) // SAFE: HMAC-SHA256
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = "123"
	secret := os.Getenv("JWT_SECRET") // key from env, not hardcoded
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func safeJWTParse(r *http.Request) {
	tokenStr := r.Header.Get("Authorization")
	// SAFE: jwt.Parse with key function validates signature
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return
	}
}

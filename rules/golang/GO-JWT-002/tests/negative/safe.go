// GO-JWT-002 negative test cases — NONE should be detected
package main

import (
	"fmt"
	"net/http"
	"os"

	jwt "github.com/golang-jwt/jwt/v5"
)

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

func safeJWTParseWithClaims(r *http.Request) {
	tokenStr := r.Header.Get("Authorization")
	var claims jwt.MapClaims
	// SAFE: ParseWithClaims validates signature
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return
	}
}

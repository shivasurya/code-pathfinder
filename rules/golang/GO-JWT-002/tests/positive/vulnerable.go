// GO-JWT-002 positive test cases — all SHOULD be detected
package main

import (
	"net/http"

	jwt "github.com/golang-jwt/jwt/v5"
)

func jwtParseUnverified(r *http.Request) {
	tokenStr := r.Header.Get("Authorization")
	parser := jwt.NewParser()
	// SINK: ParseUnverified skips signature check — forged tokens accepted
	token, _, _ := parser.ParseUnverified(tokenStr, &jwt.MapClaims{})
	_ = token
}

func jwtParseUnverifiedRaw() {
	tokenStr := "eyJhbGciOiJub25lIn0.eyJhZG1pbiI6dHJ1ZX0."
	var claims jwt.MapClaims
	parser := jwt.NewParser()
	// SINK: signature not checked at all
	tok, _, _ := parser.ParseUnverified(tokenStr, &claims)
	_ = tok
}

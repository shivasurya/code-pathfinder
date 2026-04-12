// Crypto and JWT positive test cases — all of these SHOULD be detected
package main

import (
	"crypto/des"
	"crypto/md5"
	"crypto/rc4"
	"crypto/sha1"

	jwt "github.com/golang-jwt/jwt/v5"
	"net/http"
)

// GO-CRYPTO-001: MD5 weak hash

func weakMD5New() []byte {
	h := md5.New() // SINK: MD5 is collision-broken
	h.Write([]byte("data"))
	return h.Sum(nil)
}

func weakMD5Sum() {
	data := []byte("important data")
	hash := md5.Sum(data) // SINK: MD5 weak hash
	_ = hash
}

// GO-CRYPTO-002: SHA1 weak hash

func weakSHA1New() {
	h := sha1.New() // SINK: SHA1 collision attack known
	h.Write([]byte("data"))
}

func weakSHA1Sum() {
	hash := sha1.Sum([]byte("data")) // SINK: SHA1 weak hash
	_ = hash
}

// GO-CRYPTO-003: DES cipher

func brokenDES() {
	key := []byte("12345678")
	_, _ = des.NewCipher(key) // SINK: DES 56-bit key, brute-forceable
}

func brokenTripleDES() {
	key := make([]byte, 24)
	_, _ = des.NewTripleDESCipher(key) // SINK: 3DES deprecated
}

// GO-CRYPTO-004: RC4 cipher

func brokenRC4() {
	key := []byte("secret")
	_, _ = rc4.NewCipher(key) // SINK: RC4 banned in TLS
}

// GO-JWT-001: JWT none algorithm

func jwtNoneAlgorithm() string {
	token := jwt.New(jwt.SigningMethodNone) // SINK: no crypto signature
	claims := token.Claims.(jwt.MapClaims)
	claims["admin"] = true
	signed, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType) // SINK: forged token
	return signed
}

func jwtUnsafeAllowNone() {
	token := jwt.New(jwt.SigningMethodNone)
	// SINK: UnsafeAllowNoneSignatureType explicitly bypasses signing
	token.SignedString(jwt.UnsafeAllowNoneSignatureType)
}

// GO-JWT-002: JWT ParseUnverified

func jwtParseUnverified(r *http.Request) {
	tokenStr := r.Header.Get("Authorization")
	parser := jwt.NewParser()
	// SINK: ParseUnverified skips signature check — forged tokens accepted
	token, _, _ := parser.ParseUnverified(tokenStr, &jwt.MapClaims{})
	_ = token
}

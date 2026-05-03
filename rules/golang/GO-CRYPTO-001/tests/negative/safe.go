// GO-CRYPTO-001 negative test cases — NONE should be detected
package main

import (
	"crypto/sha256"
	"crypto/sha512"
)

func safeSHA256() {
	h := sha256.New()       // SAFE: collision-resistant
	h.Write([]byte("data"))
	_ = h.Sum(nil)
}

func safeSHA512() {
	hash := sha512.Sum512([]byte("data")) // SAFE
	_ = hash
}

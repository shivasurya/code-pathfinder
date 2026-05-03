// GO-CRYPTO-002 negative test cases — NONE should be detected
package main

import (
	"crypto/hmac"
	"crypto/sha256"
)

func safeSHA256Hash() {
	h := sha256.New() // SAFE
	h.Write([]byte("data"))
	_ = h.Sum(nil)
}

func safeHMACWithSHA256() {
	mac := hmac.New(sha256.New, []byte("key")) // SAFE: HMAC-SHA256
	mac.Write([]byte("message"))
}

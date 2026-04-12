// GO-CRYPTO-003 negative test cases — NONE should be detected
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func safeAESGCM() {
	key := make([]byte, 32)
	io.ReadFull(rand.Reader, key)
	block, _ := aes.NewCipher(key)   // SAFE: AES-256
	gcm, _ := cipher.NewGCM(block)   // SAFE: GCM mode
	_ = gcm
}

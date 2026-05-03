// GO-CRYPTO-004 negative test cases — NONE should be detected
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func safeAESInsteadOfRC4() {
	key := make([]byte, 32)
	io.ReadFull(rand.Reader, key)
	block, _ := aes.NewCipher(key)
	stream := cipher.NewCTR(block, make([]byte, aes.BlockSize)) // SAFE: AES-CTR
	_ = stream
}

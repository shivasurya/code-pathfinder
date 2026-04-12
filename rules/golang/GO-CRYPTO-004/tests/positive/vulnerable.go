// GO-CRYPTO-004 positive test cases — all SHOULD be detected
package main

import "crypto/rc4"

func brokenRC4Cipher() {
	key := []byte("secretkey")
	_, _ = rc4.NewCipher(key) // SINK: RC4 banned in TLS
}

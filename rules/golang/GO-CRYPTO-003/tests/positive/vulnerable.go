// GO-CRYPTO-003 positive test cases — all SHOULD be detected
package main

import "crypto/des"

func brokenDES() {
	key := []byte("12345678")
	_, _ = des.NewCipher(key)          // SINK: DES 56-bit key, brute-forceable
}

func brokenTripleDES() {
	key := make([]byte, 24)
	_, _ = des.NewTripleDESCipher(key) // SINK: 3DES deprecated
}

// GO-CRYPTO-002 positive test cases — all SHOULD be detected
package main

import "crypto/sha1"

func weakSHA1New() {
	h := sha1.New()           // SINK: SHA1 collision attack known
	h.Write([]byte("data"))
	_ = h.Sum(nil)
}

func weakSHA1Sum() {
	hash := sha1.Sum([]byte("data")) // SINK: SHA1 weak hash
	_ = hash
}

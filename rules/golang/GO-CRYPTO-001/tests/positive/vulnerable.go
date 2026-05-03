// GO-CRYPTO-001 positive test cases — all SHOULD be detected
package main

import "crypto/md5"

func weakMD5New() []byte {
	h := md5.New()            // SINK: MD5 is collision-broken
	h.Write([]byte("data"))
	return h.Sum(nil)
}

func weakMD5Sum() {
	data := []byte("important data")
	hash := md5.Sum(data)    // SINK: MD5 weak hash
	_ = hash
}

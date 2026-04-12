// GO-CRYPTO-005 positive test cases — all SHOULD be detected
package main

import (
	"crypto/md5"
	"fmt"
)

// savePassword stores a hashed password — name matches *password* sink pattern
func savePassword(hash string) {
	fmt.Println("stored:", hash)
}

// storeUserPassword demonstrates MD5 output flowing into a password-named function
func storeUserPassword(username, plaintext string) {
	sum := md5.Sum([]byte(plaintext))         // SOURCE: md5 hash output
	savePassword(fmt.Sprintf("%x", sum))      // SINK: md5 result flows into password func
}

// checkPassword matches *password* sink pattern
func checkPassword(hash string) bool {
	return hash == "d8578edf8458ce06fbc5bb76a58c5ca4"
}

// loginUser — MD5 output flows into checkPassword
func loginUser(password string) bool {
	sum := md5.Sum([]byte(password))          // SOURCE: md5 hash output
	return checkPassword(fmt.Sprintf("%x", sum)) // SINK: flows into checkPassword
}

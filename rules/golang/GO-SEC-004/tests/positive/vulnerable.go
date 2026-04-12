// GO-SEC-004 positive test cases — credential-named variables passed to functions
package main

import "database/sql"

// Credential-named variables flowing into function calls — SHOULD be detected

func hardcodedDBPassword(db *sql.DB) {
	password := "super_secret_123"             // hardcoded credential
	db.Exec("ALTER USER admin PASSWORD ?", password) // variable 'password' as arg
}

func hardcodedAPIKey() {
	apikey := "sk-1234567890abcdef"            // hardcoded API key
	makeRequest(apikey)                         // variable 'apikey' as arg
}

func hardcodedToken() {
	token := "ghp_xxxxxxxxxxxx"                // hardcoded token
	authenticate(token)                         // variable 'token' as arg
}

func hardcodedSecret() {
	secret := "my-signing-secret"              // hardcoded secret
	signData(secret)                            // variable 'secret' as arg
}

func makeRequest(key string) {}
func authenticate(tok string) {}
func signData(s string) {}

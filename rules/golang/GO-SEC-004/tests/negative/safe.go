// GO-SEC-004 negative test cases — these use env vars, not hardcoded values
package main

import "os"

func safeEnvPassword() {
	password := os.Getenv("DB_PASSWORD") // SAFE: from environment
	_ = password
}

func safeEnvAPIKey() {
	apikey := os.Getenv("API_KEY") // SAFE: from environment
	_ = apikey
}

// GO-NET-001 positive test cases — all SHOULD be detected
package main

import "net/http"

func insecureHTTPServer() {
	mux := http.NewServeMux()
	// SINK: plaintext HTTP — credentials exposed to network
	http.ListenAndServe(":8080", mux)
}

func insecureHTTPServerNilHandler() {
	// SINK: nil handler uses DefaultServeMux
	http.ListenAndServe(":9090", nil)
}

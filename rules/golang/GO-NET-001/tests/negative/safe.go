// GO-NET-001 negative test cases — NONE should be detected
package main

import (
	"crypto/tls"
	"net/http"
)

func safeTLSServer() {
	// SAFE: ListenAndServeTLS uses TLS
	http.ListenAndServeTLS(":443", "cert.pem", "key.pem", nil)
}

func safeTLSServerWithConfig() {
	server := &http.Server{
		Addr: ":443",
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
		},
	}
	server.ListenAndServeTLS("cert.pem", "key.pem") // SAFE
}

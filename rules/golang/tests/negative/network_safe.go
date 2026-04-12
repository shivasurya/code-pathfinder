// Network security negative test cases — NONE should be detected
package main

import (
	"crypto/tls"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// SAFE: HTTPS with TLS

func safeTLSServer() {
	mux := http.NewServeMux()
	// SAFE: ListenAndServeTLS uses TLS
	http.ListenAndServeTLS(":443", "cert.pem", "key.pem", mux)
}

func safeTLSServerWithConfig() {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}
	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
	}
	server.ListenAndServeTLS("cert.pem", "key.pem") // SAFE
}

// SAFE: gRPC with TLS credentials

func safeGRPCWithTLS() {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	creds := credentials.NewTLS(tlsConfig)
	conn, _ := grpc.Dial("server:443", grpc.WithTransportCredentials(creds))
	defer conn.Close()
}

func safeGRPCWithClientCertificate() {
	creds, _ := credentials.NewClientTLSFromFile("ca.pem", "")
	conn, _ := grpc.Dial("server:443", grpc.WithTransportCredentials(creds))
	defer conn.Close()
}

// SAFE: Redirect to allowlisted path only

func safeRedirectLocalOnly(w http.ResponseWriter, r *http.Request) {
	next := r.FormValue("next")
	// SAFE: validate that next is a relative path (no scheme, no //)
	if next == "" || next[0] != '/' || (len(next) > 1 && next[1] == '/') {
		next = "/"
	}
	http.Redirect(w, r, next, http.StatusFound)
}

// SAFE: Redirect to hardcoded path

func safeRedirectHardcoded(w http.ResponseWriter, r *http.Request) {
	// SAFE: destination is a compile-time constant
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

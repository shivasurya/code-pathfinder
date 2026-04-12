// Network security positive test cases — all SHOULD be detected
package main

import (
	"net/http"

	"google.golang.org/grpc"
)

// GO-NET-001: HTTP without TLS

func insecureHTTPServer() {
	mux := http.NewServeMux()
	// SINK: plaintext HTTP — credentials exposed to network
	http.ListenAndServe(":8080", mux)
}

func insecureHTTPServerWithHandler() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	// SINK: all HTTP traffic unencrypted
	http.ListenAndServe("0.0.0.0:8080", handler)
}

// GO-NET-002: gRPC insecure connection

func grpcInsecureClient() {
	addr := "localhost:50051"
	// SINK: WithInsecure disables all transport security
	conn, _ := grpc.Dial(addr, grpc.WithInsecure())
	defer conn.Close()
}

func grpcWithInsecureOption() {
	// SINK: grpc.WithInsecure() used as option
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
	}
	conn, _ := grpc.Dial("server:443", opts...)
	defer conn.Close()
}

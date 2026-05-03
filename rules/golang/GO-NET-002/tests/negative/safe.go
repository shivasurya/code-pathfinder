// GO-NET-002 negative test cases — NONE should be detected
package main

import (
	"crypto/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func safeGRPCWithTLS() {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	creds := credentials.NewTLS(tlsConfig)
	conn, _ := grpc.Dial("service:443", grpc.WithTransportCredentials(creds))
	defer conn.Close()
}

func safeGRPCClientCert() {
	creds, _ := credentials.NewClientTLSFromFile("ca.pem", "")
	conn, _ := grpc.Dial("service:443", grpc.WithTransportCredentials(creds))
	defer conn.Close()
}

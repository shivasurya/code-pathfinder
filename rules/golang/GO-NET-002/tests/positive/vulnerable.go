// GO-NET-002 positive test cases — all SHOULD be detected
package main

import "google.golang.org/grpc"

func grpcInsecureClient() {
	addr := "backend:50051"
	// SINK: WithInsecure disables all transport security
	conn, _ := grpc.Dial(addr, grpc.WithInsecure())
	defer conn.Close()
}

func grpcInsecureOption() {
	// SINK: grpc.WithInsecure() as standalone option
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
	}
	conn, _ := grpc.Dial("service:9090", opts...)
	defer conn.Close()
}

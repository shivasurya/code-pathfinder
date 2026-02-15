package main

import (
	"fmt"
	"net/http"
)

const DefaultPort = 8080

var version = "1.0.0"

func main() {
	http.HandleFunc("/", handleIndex)
	addr := fmt.Sprintf(":%d", DefaultPort)
	http.ListenAndServe(addr, nil)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "version: %s", version)
}

package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func mains() {
	address := "127.0.0.1:5036"

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()
	fmt.Println("Connected to the server. Type your JSON message and press enter.")

	// Start a goroutine to listen for messages from the server
	go func() {
		serverReader := bufio.NewReader(conn)
		for {
			serverMessage, err := serverReader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading from server:", err)
				return
			}
			if serverMessage != "" {
				fmt.Printf("Message from server: %s\n", serverMessage)
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter JSON message: ")
		jsonMessage, _ := reader.ReadString('\n')
		jsonMessage = strings.TrimSpace(jsonMessage) // Trim newline character for clean input

		// Calculate the content length
		contentLength := len([]byte(jsonMessage))

		// Create the full message with headers
		fullMessage := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", contentLength, jsonMessage)

		// Send the message
		_, err := conn.Write([]byte(fullMessage))
		if err != nil {
			fmt.Println("Error sending message:", err)
			return
		}

		fmt.Println("Message sent. Type another message, or CTRL+C to quit.")
	}
}

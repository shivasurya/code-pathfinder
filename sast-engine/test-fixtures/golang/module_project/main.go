package main

import (
	"fmt"
	"github.com/example/testapp/handlers"
	"github.com/example/testapp/models"
)

func main() {
	server := models.NewServer("localhost", 8080)
	handlers.RegisterRoutes(server)
	fmt.Println("Server started")
}

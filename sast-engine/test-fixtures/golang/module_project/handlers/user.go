package handlers

import (
	"encoding/json"
	"net/http"
	m "github.com/example/testapp/models"
)

func GetUser(w http.ResponseWriter, r *http.Request) {
	user := m.User{ID: 1, Name: "Alice"}
	json.NewEncoder(w).Encode(user)
}

func RegisterRoutes(server *m.Server) {
	// Registration logic
}

package utils

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// LogRequestDuration logs the duration of a request
func LogRequestDuration(handler string, start time.Time) {
	log.Printf("%s took %v", handler, time.Since(start))
}

// ValidateMethod checks if the request method matches the expected method
func ValidateMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

// DecodeJSONRequest decodes a JSON request body into a struct
func DecodeJSONRequest(w http.ResponseWriter, r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return err
	}
	return nil
}

// SendJSONResponse sends a JSON response
func SendJSONResponse(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// SendErrorResponse sends an error response
func SendErrorResponse(w http.ResponseWriter, message string, err error) {
	log.Printf("Error: %s: %v", message, err)
	http.Error(w, message, http.StatusInternalServerError)
}

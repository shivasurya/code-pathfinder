package handlers

import (
	"fmt"
	"net/http"
	"github.com/example/testapp/utils/validation"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if !validation.ValidateToken(token) {
			http.Error(w, "Unauthorized", 401)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func Login(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Login handler")
}

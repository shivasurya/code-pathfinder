package validation

import "strings"

func ValidateToken(token string) bool {
	return strings.HasPrefix(token, "Bearer ")
}

func ValidateEmail(email string) bool {
	return strings.Contains(email, "@")
}

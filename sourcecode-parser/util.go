package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func GenerateMethodID(methodName string, parameters []string, sourceFile string) string {
	hashInput := fmt.Sprintf("%s-%s-%s", methodName, parameters, sourceFile)
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

func GenerateSha256(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

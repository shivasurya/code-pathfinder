package java

import (
	"path/filepath"
	"strings"
)

func ExtractVisibilityModifier(modifiers string) string {
	words := strings.Fields(modifiers)
	for _, word := range words {
		switch word {
		case "public", "private", "protected":
			return word
		}
	}
	return "" // return an empty string if no visibility modifier is found
}

func IsJavaSourceFile(filename string) bool {
	return filepath.Ext(filename) == ".java"
}

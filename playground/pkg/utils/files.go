package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const (
	// FilePermissions for created files
	FilePermissions = 0644
)

// CreateTempWorkspace creates a temporary directory for analysis
func CreateTempWorkspace(prefix string) (string, error) {
	// Create a unique directory name
	dirName := fmt.Sprintf("%s-%s", prefix, uuid.New().String())
	tmpDir := filepath.Join(os.TempDir(), dirName)

	// Create directory with secure permissions
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	return tmpDir, nil
}

// WriteSourceAndQueryFiles writes the source and query files to the workspace
func WriteSourceAndQueryFiles(dir, source, query string) error {
	if err := WriteSourceFile(dir, source); err != nil {
		return err
	}
	return WriteFile(filepath.Join(dir, "query.ql"), query)
}

// WriteSourceFile writes the Java source code to a file
func WriteSourceFile(dir, source string) error {
	return WriteFile(filepath.Join(dir, "Main.java"), source)
}

// WriteFile writes content to a file with proper permissions
func WriteFile(path, content string) error {
	// Create or truncate the file
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, FilePermissions)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Write content
	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("failed to write content: %v", err)
	}

	return nil
}

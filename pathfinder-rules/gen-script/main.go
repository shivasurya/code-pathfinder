package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type CQLFileContent struct {
	FileName string `json:"file_name"`
	Content  string `json:"content"`
}

type CQLFiles struct {
	Directory string           `json:"ruleset"`
	Files     []CQLFileContent `json:"files"`
}

func main() {
	rootDir := "../../pathfinder-rules" // Set the root directory to start traversal
	err := filepath.Walk(rootDir, processDirectory)
	if err != nil {
		fmt.Printf("Error walking the path %q: %v\n", rootDir, err)
	}
}

func processDirectory(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if info.IsDir() {
		var cqlFiles []CQLFileContent
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".cql" {
				filePath := filepath.Join(path, entry.Name())
				content, err := os.ReadFile(filePath)
				if err != nil {
					return err
				}
				cqlFiles = append(cqlFiles, CQLFileContent{
					FileName: entry.Name(),
					Content:  string(content),
				})
			}
		}

		if len(cqlFiles) > 0 {
			jsonData := CQLFiles{
				Directory: filepath.Base(path),
				Files:     cqlFiles,
			}

			jsonFileName := filepath.Base(path) + ".json"
			jsonFilePath := filepath.Join(path, jsonFileName)
			jsonBytes, err := json.MarshalIndent(jsonData, "", "  ")
			if err != nil {
				return err
			}

			err = os.WriteFile(jsonFilePath, jsonBytes, 0644)
			if err != nil {
				return err
			}

			fmt.Printf("Created JSON file at: %s\n", jsonFilePath)
		}
	}

	return nil
}

package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/model"
	sitter "github.com/smacker/go-tree-sitter"
)

var verboseFlag bool

// GenerateMethodID generates a unique SHA256 hash ID for a method.
func GenerateMethodID(methodName string, parameters []string, sourceFile string, lineNumber ...uint32) string {
	// Include line number in hash to ensure uniqueness for methods/constructors
	// with same name/params in same file (e.g., multiple __init__ in one file).
	lineStr := ""
	if len(lineNumber) > 0 {
		lineStr = fmt.Sprintf(":%d", lineNumber[0])
	}
	hashInput := fmt.Sprintf("%s-%s-%s%s", methodName, parameters, sourceFile, lineStr)
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

// GenerateSha256 generates a SHA256 hash from an input string.
func GenerateSha256(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// appendUnique appends a node to a slice only if it's not already present.
func appendUnique(slice []*Node, node *Node) []*Node {
	if slices.Contains(slice, node) {
		return slice
	}
	return append(slice, node)
}

// FormatType formats various types to string representation.
func FormatType(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%.2f", val)
	case []any:
		//nolint:all
		jsonBytes, _ := json.Marshal(val)
		return string(jsonBytes)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// EnableVerboseLogging enables verbose logging mode.
func EnableVerboseLogging() {
	verboseFlag = true
}

// Log logs a message if verbose logging is enabled.
func Log(message string, args ...any) {
	if verboseFlag {
		log.Println(message, args)
	}
}

// Fmt prints formatted output if verbose logging is enabled.
func Fmt(format string, args ...any) {
	if verboseFlag {
		fmt.Printf(format, args...)
	}
}

// IsGitHubActions checks if running in GitHub Actions environment.
func IsGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

// extractVisibilityModifier extracts visibility modifier from a string of modifiers.
func extractVisibilityModifier(modifiers string) string {
	words := strings.FieldsSeq(modifiers)
	for word := range words {
		switch word {
		case "public", "private", "protected":
			return word
		}
	}
	return "" // return an empty string if no visibility modifier is found
}

// isJavaSourceFile checks if a file is a Java source file.
func isJavaSourceFile(filename string) bool {
	return filepath.Ext(filename) == ".java"
}

// isPythonSourceFile checks if a file is a Python source file.
func isPythonSourceFile(filename string) bool {
	return filepath.Ext(filename) == ".py"
}

// isGoSourceFile checks if a file is a Go source file.
func isGoSourceFile(filename string) bool {
	return filepath.Ext(filename) == ".go"
}

//nolint:all
func hasAccess(node *sitter.Node, variableName string, sourceCode []byte) bool {
	if node == nil {
		return false
	}
	if node.Type() == "identifier" && node.Content(sourceCode) == variableName {
		return true
	}

	// Recursively check all children of the current node
	for i := 0; i < int(node.ChildCount()); i++ {
		childNode := node.Child(i)
		if hasAccess(childNode, variableName, sourceCode) {
			return true
		}
	}

	// Continue checking in the next sibling
	return hasAccess(node.NextSibling(), variableName, sourceCode)
}

// parseJavadocTags parses Javadoc tags from comment content.
func parseJavadocTags(commentContent string) *model.Javadoc {
	javaDoc := &model.Javadoc{}
	var javadocTags []*model.JavadocTag

	commentLines := strings.Split(commentContent, "\n")
	for _, line := range commentLines {
		line = strings.TrimSpace(line)
		// line may start with /** or *
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "@") {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				tagName := strings.TrimPrefix(parts[0], "@")
				tagText := strings.TrimSpace(parts[1])

				var javadocTag *model.JavadocTag
				switch tagName {
				case "author":
					javadocTag = model.NewJavadocTag(tagName, tagText, "author")
					javaDoc.Author = tagText
				case "param":
					javadocTag = model.NewJavadocTag(tagName, tagText, "param")
				case "see":
					javadocTag = model.NewJavadocTag(tagName, tagText, "see")
				case "throws":
					javadocTag = model.NewJavadocTag(tagName, tagText, "throws")
				case "version":
					javadocTag = model.NewJavadocTag(tagName, tagText, "version")
					javaDoc.Version = tagText
				case "since":
					javadocTag = model.NewJavadocTag(tagName, tagText, "since")
				default:
					javadocTag = model.NewJavadocTag(tagName, tagText, "unknown")
				}
				javadocTags = append(javadocTags, javadocTag)
			}
		}
	}

	javaDoc.Tags = javadocTags
	javaDoc.NumberOfCommentLines = len(commentLines)
	javaDoc.CommentedCodeElements = commentContent

	return javaDoc
}

//nolint:all
func extractMethodName(node *sitter.Node, sourceCode []byte, filepath string) (string, string) {
	var methodID string

	// if the child node is method_declaration, extract method name, modifiers, parameters, and return type
	var methodName string
	var modifiers, parameters []string

	if node.Type() == "method_declaration" {
		// Iterate over all children of the method_declaration node
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			switch child.Type() {
			case "modifiers", "marker_annotation", "annotation":
				// This child is a modifier or annotation, add its content to modifiers
				modifiers = append(modifiers, child.Content(sourceCode)) //nolint:all
			case "identifier":
				// This child is the method name
				methodName = child.Content(sourceCode)
			case "formal_parameters":
				// This child represents formal parameters; iterate through its children
				for j := 0; j < int(child.NamedChildCount()); j++ {
					param := child.NamedChild(j)
					parameters = append(parameters, param.Content(sourceCode))
				}
			}
		}
	}

	// check if type is method_invocation
	// if the child node is method_invocation, extract method name
	if node.Type() == "method_invocation" {
		for j := 0; j < int(node.ChildCount()); j++ {
			child := node.Child(j)
			if child.Type() == "identifier" {
				if methodName == "" {
					methodName = child.Content(sourceCode)
				} else {
					methodName = methodName + "." + child.Content(sourceCode)
				}
			}

			argumentsNode := node.ChildByFieldName("argument_list")
			// add data type of arguments list
			if argumentsNode != nil {
				for k := 0; k < int(argumentsNode.ChildCount()); k++ {
					argument := argumentsNode.Child(k)
					parameters = append(parameters, argument.Child(0).Content(sourceCode))
				}
			}

		}
	}
	content := node.Content(sourceCode)
	lineNumber := int(node.StartPoint().Row) + 1
	columnNumber := int(node.StartPoint().Column) + 1
	// convert to string and merge
	content += " " + strconv.Itoa(lineNumber) + ":" + strconv.Itoa(columnNumber)

	// Prefix method declarations to avoid ID collisions with invocations
	prefix := ""
	if node.Type() == "method_declaration" {
		prefix = "method:"
	}

	methodID = GenerateMethodID(prefix+methodName, parameters, filepath+"/"+content)
	return methodName, methodID
}

// getFiles walks through a directory and returns all source files (Java, Python, Go, Dockerfile, docker-compose).
// It skips vendor/, testdata/, node_modules/, .git/, and directories starting with "_".
func getFiles(directory string) ([]string, error) {
	var files []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip directories that should never be scanned
		if info.IsDir() {
			name := info.Name()
			switch name {
			case "vendor", "testdata", "node_modules", ".git":
				return filepath.SkipDir
			}
			if strings.HasPrefix(name, "_") {
				return filepath.SkipDir
			}
			return nil
		}
		// append java, python, go, dockerfile, and docker-compose files
		ext := filepath.Ext(path)
		base := filepath.Base(path)
		baseLower := strings.ToLower(base)

		switch {
		case ext == ".java" || ext == ".py" || ext == ".go":
			files = append(files, path)
		case strings.HasPrefix(baseLower, "dockerfile"):
			// Match Dockerfile, Dockerfile.dev, dockerfile, etc.
			files = append(files, path)
		case strings.Contains(baseLower, "docker-compose") && (ext == ".yml" || ext == ".yaml"):
			// Match docker-compose.yml, docker-compose.yaml, etc.
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// readFile reads the contents of a file.
func readFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return content, nil
}

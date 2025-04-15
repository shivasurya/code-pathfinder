package java

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/stretchr/testify/assert"
)

// TestParseJavadocTags tests the ParseJavadocTags function
func TestParseJavadocTags(t *testing.T) {
	t.Run("Basic javadoc with author tag", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`/**
 * This is a simple class description
 * @author John Doe
 */`)

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Call the function with our parsed node
		javadoc := ParseJavadocTags(rootNode, sourceCode)

		// Assertions
		assert.NotNil(t, javadoc)
		assert.Equal(t, 1, len(javadoc.Tags))
		assert.Equal(t, "author", javadoc.Tags[0].TagName)
		assert.Equal(t, "John Doe", javadoc.Tags[0].Text)
		assert.Equal(t, "author", javadoc.Tags[0].DocType)
		assert.Equal(t, "John Doe", javadoc.Author)
		assert.Equal(t, 4, javadoc.NumberOfCommentLines)
		assert.Equal(t, string(sourceCode), javadoc.CommentedCodeElements)
	})

	t.Run("Javadoc with multiple tags", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`/**
 * This is a class with multiple javadoc tags
 * @author Jane Smith
 * @version 1.0.0
 * @since 2023-01-01
 */`)

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Call the function with our parsed node
		javadoc := ParseJavadocTags(rootNode, sourceCode)

		// Assertions
		assert.NotNil(t, javadoc)
		assert.Equal(t, 3, len(javadoc.Tags))

		// Check author tag
		assert.Equal(t, "author", javadoc.Tags[0].TagName)
		assert.Equal(t, "Jane Smith", javadoc.Tags[0].Text)
		assert.Equal(t, "Jane Smith", javadoc.Author)

		// Check version tag
		assert.Equal(t, "version", javadoc.Tags[1].TagName)
		assert.Equal(t, "1.0.0", javadoc.Tags[1].Text)
		assert.Equal(t, "1.0.0", javadoc.Version)

		// Check since tag
		assert.Equal(t, "since", javadoc.Tags[2].TagName)
		assert.Equal(t, "2023-01-01", javadoc.Tags[2].Text)

		// Check comment lines count
		assert.Equal(t, 6, javadoc.NumberOfCommentLines)
	})

	t.Run("Javadoc with param and throws tags", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`/**
 * Method description
 * @param input The input string to process
 * @param count The number of times to process
 * @throws IllegalArgumentException If input is invalid
 * @see OtherClass
 */`)

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Call the function with our parsed node
		javadoc := ParseJavadocTags(rootNode, sourceCode)

		// Assertions
		assert.NotNil(t, javadoc)
		assert.Equal(t, 4, len(javadoc.Tags))

		// Check param tags
		paramTags := 0
		for _, tag := range javadoc.Tags {
			if tag.TagName == "param" {
				paramTags++
				assert.Contains(t, []string{"input The input string to process", "count The number of times to process"}, tag.Text)
			}
		}
		assert.Equal(t, 2, paramTags)

		// Check throws tag
		throwsTag := false
		for _, tag := range javadoc.Tags {
			if tag.TagName == "throws" {
				throwsTag = true
				assert.Equal(t, "IllegalArgumentException If input is invalid", tag.Text)
			}
		}
		assert.True(t, throwsTag)

		// Check see tag
		seeTag := false
		for _, tag := range javadoc.Tags {
			if tag.TagName == "see" {
				seeTag = true
				assert.Equal(t, "OtherClass", tag.Text)
			}
		}
		assert.True(t, seeTag)

		// Check number of lines
		assert.Equal(t, 7, javadoc.NumberOfCommentLines)
	})

	t.Run("Empty javadoc", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`/**
 */`)

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Call the function with our parsed node
		javadoc := ParseJavadocTags(rootNode, sourceCode)

		// Assertions
		assert.NotNil(t, javadoc)
		assert.Equal(t, 0, len(javadoc.Tags))
		assert.Equal(t, 2, javadoc.NumberOfCommentLines)
		assert.Equal(t, string(sourceCode), javadoc.CommentedCodeElements)
	})

	t.Run("Javadoc with malformed tags", func(t *testing.T) {
		// Setup - tag without text
		sourceCode := []byte(`/**
 * Description
 * @author
 * @version
 */`)

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Call the function with our parsed node
		javadoc := ParseJavadocTags(rootNode, sourceCode)

		// Assertions
		assert.NotNil(t, javadoc)
		// The current implementation doesn't add tags without text values
		assert.Equal(t, 0, len(javadoc.Tags))
		assert.Equal(t, 5, javadoc.NumberOfCommentLines) // Including empty lines
		assert.Equal(t, string(sourceCode), javadoc.CommentedCodeElements)
	})

	t.Run("Javadoc with non-standard tags", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`/**
 * Description
 * @custom This is a custom tag
 * @deprecated Use newMethod() instead
 */`)

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Call the function with our parsed node
		javadoc := ParseJavadocTags(rootNode, sourceCode)

		// Assertions
		assert.NotNil(t, javadoc)
		assert.Equal(t, 2, len(javadoc.Tags))

		// Find custom tag
		customTagFound := false
		deprecatedTagFound := false

		for _, tag := range javadoc.Tags {
			if tag.TagName == "custom" {
				customTagFound = true
				assert.Equal(t, "This is a custom tag", tag.Text)
				assert.Equal(t, "unknown", tag.DocType)
			}
			if tag.TagName == "deprecated" {
				deprecatedTagFound = true
				assert.Equal(t, "Use newMethod() instead", tag.Text)
				assert.Equal(t, "unknown", tag.DocType)
			}
		}

		assert.True(t, customTagFound, "Custom tag should be found")
		assert.True(t, deprecatedTagFound, "Deprecated tag should be found")
		assert.Equal(t, 5, javadoc.NumberOfCommentLines)
	})
}

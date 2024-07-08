package model

import (
	"testing"
)

func TestNewJavadocTag(t *testing.T) {
	tagName := "author"
	text := "John Doe"
	docType := "class"

	tag := NewJavadocTag(tagName, text, docType)

	if tag.TagName != tagName {
		t.Errorf("Expected TagName to be %s, got %s", tagName, tag.TagName)
	}

	if tag.Text != text {
		t.Errorf("Expected Text to be %s, got %s", text, tag.Text)
	}

	if tag.DocType != docType {
		t.Errorf("Expected DocType to be %s, got %s", docType, tag.DocType)
	}
}

func TestNewJavadocTagWithEmptyValues(t *testing.T) {
	tag := NewJavadocTag("", "", "")

	if tag.TagName != "" {
		t.Errorf("Expected TagName to be empty, got %s", tag.TagName)
	}

	if tag.Text != "" {
		t.Errorf("Expected Text to be empty, got %s", tag.Text)
	}

	if tag.DocType != "" {
		t.Errorf("Expected DocType to be empty, got %s", tag.DocType)
	}
}

func TestJavadocTagsSlice(t *testing.T) {
	tags := []*JavadocTag{
		{TagName: "author", Text: "John Doe", DocType: "class"},
		{TagName: "version", Text: "1.0", DocType: "class"},
	}
	jdoc := &Javadoc{Tags: tags}

	if len(jdoc.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(jdoc.Tags))
	}

	if jdoc.Author != "John Doe" {
		t.Errorf("Expected Author to be 'John Doe', got '%s'", jdoc.Author)
	}

	if jdoc.Version != "1.0" {
		t.Errorf("Expected Version to be '1.0', got '%s'", jdoc.Version)
	}
}

func TestJavadocWithNoTags(t *testing.T) {
	jdoc := &Javadoc{}

	if len(jdoc.Tags) != 0 {
		t.Errorf("Expected 0 tags, got %d", len(jdoc.Tags))
	}

	if jdoc.Author != "" {
		t.Errorf("Expected Author to be empty, got '%s'", jdoc.Author)
	}

	if jdoc.Version != "" {
		t.Errorf("Expected Version to be empty, got '%s'", jdoc.Version)
	}
}

func TestJavadocWithCommentedCodeElements(t *testing.T) {
	jdoc := &Javadoc{CommentedCodeElements: "MyClass"}

	if jdoc.CommentedCodeElements != "MyClass" {
		t.Errorf("Expected CommentedCodeElements to be 'MyClass', got '%s'", jdoc.CommentedCodeElements)
	}
}

func TestJavadocWithNumberOfCommentLines(t *testing.T) {
	jdoc := &Javadoc{NumberOfCommentLines: 5}

	if jdoc.NumberOfCommentLines != 5 {
		t.Errorf("Expected NumberOfCommentLines to be 5, got %d", jdoc.NumberOfCommentLines)
	}
}

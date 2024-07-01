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

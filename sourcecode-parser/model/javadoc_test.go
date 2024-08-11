package model

import (
	"reflect"
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
		{TagName: "author", Text: "John Doe", DocType: "author"},
		{TagName: "version", Text: "1.0", DocType: "version"},
	}
	jdoc := &Javadoc{Tags: tags}
	jdoc.Author = "John Doe"
	jdoc.Version = "1.0"

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

func TestGetCommentAuthor(t *testing.T) {
	tests := []struct {
		name     string
		javadoc  *Javadoc
		expected string
	}{
		{
			name: "Single author tag",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "author", Text: "Jane Doe", DocType: "author"},
				},
			},
			expected: "Jane Doe",
		},
		{
			name: "Multiple author tags",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "author", Text: "John Smith", DocType: "author"},
					{TagName: "author", Text: "Jane Doe", DocType: "author"},
				},
			},
			expected: "John Smith",
		},
		{
			name: "No author tag",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "version", Text: "1.0", DocType: "version"},
				},
			},
			expected: "",
		},
		{
			name:     "Empty Javadoc",
			javadoc:  &Javadoc{},
			expected: "",
		},
		{
			name: "Author tag with empty text",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "author", Text: "", DocType: "author"},
				},
			},
			expected: "",
		},
		{
			name: "Author tag not first in list",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "version", Text: "1.0", DocType: "version"},
					{TagName: "author", Text: "Alice Cooper", DocType: "author"},
				},
			},
			expected: "Alice Cooper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.javadoc.GetCommentAuthor()
			if result != tt.expected {
				t.Errorf("GetCommentAuthor() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetCommentSee(t *testing.T) {
	tests := []struct {
		name     string
		javadoc  *Javadoc
		expected string
	}{
		{
			name: "Single see tag",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "see", Text: "com.example.OtherClass", DocType: "see"},
				},
			},
			expected: "com.example.OtherClass",
		},
		{
			name: "Multiple see tags",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "see", Text: "com.example.FirstClass", DocType: "see"},
					{TagName: "see", Text: "com.example.SecondClass", DocType: "see"},
				},
			},
			expected: "com.example.FirstClass",
		},
		{
			name: "No see tag",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "param", Text: "input", DocType: "param"},
				},
			},
			expected: "",
		},
		{
			name:     "Empty Javadoc",
			javadoc:  &Javadoc{},
			expected: "",
		},
		{
			name: "See tag with empty text",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "see", Text: "", DocType: "see"},
				},
			},
			expected: "",
		},
		{
			name: "See tag not first in list",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "param", Text: "input", DocType: "param"},
					{TagName: "see", Text: "com.example.ReferencedClass", DocType: "see"},
				},
			},
			expected: "com.example.ReferencedClass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.javadoc.GetCommentSee()
			if result != tt.expected {
				t.Errorf("GetCommentSee() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetCommentVersion(t *testing.T) {
	tests := []struct {
		name     string
		javadoc  *Javadoc
		expected string
	}{
		{
			name: "Single version tag",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "version", Text: "1.0.0", DocType: "version"},
				},
			},
			expected: "1.0.0",
		},
		{
			name: "Multiple version tags",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "version", Text: "1.0.0", DocType: "version"},
					{TagName: "version", Text: "2.0.0", DocType: "version"},
				},
			},
			expected: "1.0.0",
		},
		{
			name: "No version tag",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "author", Text: "John Doe", DocType: "author"},
				},
			},
			expected: "",
		},
		{
			name:     "Empty Javadoc",
			javadoc:  &Javadoc{},
			expected: "",
		},
		{
			name: "Version tag with empty text",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "version", Text: "", DocType: "version"},
				},
			},
			expected: "",
		},
		{
			name: "Version tag not first in list",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "author", Text: "Jane Smith", DocType: "author"},
					{TagName: "version", Text: "3.1.4", DocType: "version"},
				},
			},
			expected: "3.1.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.javadoc.GetCommentVersion()
			if result != tt.expected {
				t.Errorf("GetCommentVersion() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetCommentSince(t *testing.T) {
	tests := []struct {
		name     string
		javadoc  *Javadoc
		expected string
	}{
		{
			name: "Single since tag",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "since", Text: "1.5", DocType: "since"},
				},
			},
			expected: "1.5",
		},
		{
			name: "Multiple since tags",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "since", Text: "1.0", DocType: "since"},
					{TagName: "since", Text: "2.0", DocType: "since"},
				},
			},
			expected: "1.0",
		},
		{
			name: "No since tag",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "param", Text: "input", DocType: "param"},
				},
			},
			expected: "",
		},
		{
			name:     "Empty Javadoc",
			javadoc:  &Javadoc{},
			expected: "",
		},
		{
			name: "Since tag with empty text",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "since", Text: "", DocType: "since"},
				},
			},
			expected: "",
		},
		{
			name: "Since tag not first in list",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "param", Text: "input", DocType: "param"},
					{TagName: "since", Text: "3.0", DocType: "since"},
				},
			},
			expected: "3.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.javadoc.GetCommentSince()
			if result != tt.expected {
				t.Errorf("GetCommentSince() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetCommentParam(t *testing.T) {
	tests := []struct {
		name     string
		javadoc  *Javadoc
		expected []string
	}{
		{
			name: "Single param tag",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "param", Text: "input The input string", DocType: "param"},
				},
			},
			expected: []string{"input The input string"},
		},
		{
			name: "Multiple param tags",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "param", Text: "a First parameter", DocType: "param"},
					{TagName: "param", Text: "b Second parameter", DocType: "param"},
					{TagName: "param", Text: "c Third parameter", DocType: "param"},
				},
			},
			expected: []string{"a First parameter", "b Second parameter", "c Third parameter"},
		},
		{
			name: "Mixed tags with param",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "author", Text: "John Doe", DocType: "author"},
					{TagName: "param", Text: "x Parameter x", DocType: "param"},
					{TagName: "return", Text: "The result", DocType: "return"},
					{TagName: "param", Text: "y Parameter y", DocType: "param"},
				},
			},
			expected: []string{"x Parameter x", "y Parameter y"},
		},
		{
			name: "No param tags",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "author", Text: "Jane Smith", DocType: "author"},
					{TagName: "version", Text: "1.0", DocType: "version"},
				},
			},
			expected: []string{},
		},
		{
			name:     "Empty Javadoc",
			javadoc:  &Javadoc{},
			expected: []string{},
		},
		{
			name: "Param tag with empty text",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "param", Text: "", DocType: "param"},
				},
			},
			expected: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.javadoc.GetCommentParam()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("GetCommentParam() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetCommentThrows(t *testing.T) {
	tests := []struct {
		name     string
		javadoc  *Javadoc
		expected string
	}{
		{
			name: "Single throws tag",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "throws", Text: "IOException If an I/O error occurs", DocType: "throws"},
				},
			},
			expected: "IOException If an I/O error occurs",
		},
		{
			name: "Multiple throws tags",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "throws", Text: "IllegalArgumentException If the argument is invalid", DocType: "throws"},
					{TagName: "throws", Text: "NullPointerException If the input is null", DocType: "throws"},
				},
			},
			expected: "IllegalArgumentException If the argument is invalid",
		},
		{
			name: "No throws tag",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "param", Text: "input The input string", DocType: "param"},
					{TagName: "return", Text: "The processed result", DocType: "return"},
				},
			},
			expected: "",
		},
		{
			name:     "Empty Javadoc",
			javadoc:  &Javadoc{},
			expected: "",
		},
		{
			name: "Throws tag with empty text",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "throws", Text: "", DocType: "throws"},
				},
			},
			expected: "",
		},
		{
			name: "Throws tag not first in list",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "param", Text: "x The x coordinate", DocType: "param"},
					{TagName: "throws", Text: "ArithmeticException If division by zero occurs", DocType: "throws"},
				},
			},
			expected: "ArithmeticException If division by zero occurs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.javadoc.GetCommentThrows()
			if result != tt.expected {
				t.Errorf("GetCommentThrows() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetCommentReturn(t *testing.T) {
	tests := []struct {
		name     string
		javadoc  *Javadoc
		expected string
	}{
		{
			name: "Single return tag",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "return", Text: "The processed result", DocType: "return"},
				},
			},
			expected: "The processed result",
		},
		{
			name: "Multiple return tags",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "return", Text: "First return description", DocType: "return"},
					{TagName: "return", Text: "Second return description", DocType: "return"},
				},
			},
			expected: "First return description",
		},
		{
			name: "No return tag",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "param", Text: "input The input string", DocType: "param"},
					{TagName: "throws", Text: "IOException If an I/O error occurs", DocType: "throws"},
				},
			},
			expected: "",
		},
		{
			name:     "Empty Javadoc",
			javadoc:  &Javadoc{},
			expected: "",
		},
		{
			name: "Return tag with empty text",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "return", Text: "", DocType: "return"},
				},
			},
			expected: "",
		},
		{
			name: "Return tag not first in list",
			javadoc: &Javadoc{
				Tags: []*JavadocTag{
					{TagName: "param", Text: "x The x coordinate", DocType: "param"},
					{TagName: "return", Text: "The calculated result", DocType: "return"},
				},
			},
			expected: "The calculated result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.javadoc.GetCommentReturn()
			if result != tt.expected {
				t.Errorf("GetCommentReturn() = %v, want %v", result, tt.expected)
			}
		})
	}
}

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewJavadocTag(t *testing.T) {
	t.Run("Basic tag creation", func(t *testing.T) {
		tag := NewJavadocTag("param", "description", "method")
		assert.Equal(t, "param", tag.TagName)
		assert.Equal(t, "description", tag.Text)
		assert.Equal(t, "method", tag.DocType)
	})

	t.Run("Empty tag creation", func(t *testing.T) {
		tag := NewJavadocTag("", "", "")
		assert.Equal(t, "", tag.TagName)
		assert.Equal(t, "", tag.Text)
		assert.Equal(t, "", tag.DocType)
	})
}

func TestJavadoc(t *testing.T) {
	// Setup some test tags
	tags := []*JavadocTag{
		NewJavadocTag("author", "John Doe", "class"),
		NewJavadocTag("version", "1.0", "class"),
		NewJavadocTag("see", "OtherClass", "method"),
		NewJavadocTag("since", "2.0", "class"),
		NewJavadocTag("param", "arg1 - first argument", "method"),
		NewJavadocTag("param", "arg2 - second argument", "method"),
		NewJavadocTag("throws", "IllegalArgumentException", "method"),
		NewJavadocTag("return", "computed value", "method"),
	}

	javadoc := &Javadoc{
		Tags:                  tags,
		NumberOfCommentLines:  10,
		CommentedCodeElements: "/** Test javadoc */",
		Version:               "1.0",
		Author:                "John Doe",
	}

	t.Run("GetCommentAuthor", func(t *testing.T) {
		assert.Equal(t, "John Doe", javadoc.GetCommentAuthor())

		// Test with no author tag
		emptyJavadoc := &Javadoc{Tags: []*JavadocTag{}}
		assert.Equal(t, "", emptyJavadoc.GetCommentAuthor())
	})

	t.Run("GetCommentVersion", func(t *testing.T) {
		assert.Equal(t, "1.0", javadoc.GetCommentVersion())

		// Test with no version tag
		emptyJavadoc := &Javadoc{Tags: []*JavadocTag{}}
		assert.Equal(t, "", emptyJavadoc.GetCommentVersion())
	})

	t.Run("GetCommentSee", func(t *testing.T) {
		assert.Equal(t, "OtherClass", javadoc.GetCommentSee())

		// Test with no see tag
		emptyJavadoc := &Javadoc{Tags: []*JavadocTag{}}
		assert.Equal(t, "", emptyJavadoc.GetCommentSee())
	})

	t.Run("GetCommentSince", func(t *testing.T) {
		assert.Equal(t, "2.0", javadoc.GetCommentSince())

		// Test with no since tag
		emptyJavadoc := &Javadoc{Tags: []*JavadocTag{}}
		assert.Equal(t, "", emptyJavadoc.GetCommentSince())
	})

	t.Run("GetCommentParam", func(t *testing.T) {
		params := javadoc.GetCommentParam()
		assert.Equal(t, 2, len(params))
		assert.Contains(t, params, "arg1 - first argument")
		assert.Contains(t, params, "arg2 - second argument")

		// Test with no param tags
		emptyJavadoc := &Javadoc{Tags: []*JavadocTag{}}
		assert.Empty(t, emptyJavadoc.GetCommentParam())
	})

	t.Run("GetCommentThrows", func(t *testing.T) {
		assert.Equal(t, "IllegalArgumentException", javadoc.GetCommentThrows())

		// Test with no throws tag
		emptyJavadoc := &Javadoc{Tags: []*JavadocTag{}}
		assert.Equal(t, "", emptyJavadoc.GetCommentThrows())
	})

	t.Run("GetCommentReturn", func(t *testing.T) {
		assert.Equal(t, "computed value", javadoc.GetCommentReturn())

		// Test with no return tag
		emptyJavadoc := &Javadoc{Tags: []*JavadocTag{}}
		assert.Equal(t, "", emptyJavadoc.GetCommentReturn())
	})

	t.Run("GetProxyEnv", func(t *testing.T) {
		proxyEnv := javadoc.GetProxyEnv()

		assert.Equal(t, 10, proxyEnv["GetNumberOfCommentLines"])
		assert.Equal(t, "/** Test javadoc */", proxyEnv["GetCommentedCodeElements"])
		assert.Equal(t, "John Doe", proxyEnv["GetCommentAuthor"])
		assert.Equal(t, "1.0", proxyEnv["GetCommentVersion"])
		assert.Equal(t, "computed value", proxyEnv["GetCommentReturn"])
		assert.Equal(t, "OtherClass", proxyEnv["GetCommentSee"])
		assert.Equal(t, "2.0", proxyEnv["GetCommentSince"])
		assert.Equal(t, []string{"arg1 - first argument", "arg2 - second argument"}, proxyEnv["GetCommentParam"])
		assert.Equal(t, "IllegalArgumentException", proxyEnv["GetCommentThrows"])
		assert.Equal(t, "Javadoc", proxyEnv["GetPrimaryQlClass"])
	})
}

package model

type Javadoc struct {
	Tags                  []*JavadocTag
	NumberOfCommentLines  int
	CommentedCodeElements string
	Version               string // redundant from tags
	Author                string // redundant from tags
}

// JavadocTag represents a generic Javadoc tag.
type JavadocTag struct {
	TagName string
	Text    string
	DocType string
}

// NewJavadocTag is a constructor for JavadocTag.
func NewJavadocTag(tagName, text, docType string) *JavadocTag {
	return &JavadocTag{
		TagName: tagName,
		Text:    text,
		DocType: docType,
	}
}

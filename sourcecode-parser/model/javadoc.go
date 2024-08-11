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

func (j *Javadoc) GetCommentAuthor() string {
	for _, tag := range j.Tags {
		if tag.TagName == "author" {
			return tag.Text
		}
	}
	return ""
}

func (j *Javadoc) GetCommentSee() string {
	for _, tag := range j.Tags {
		if tag.TagName == "see" {
			return tag.Text
		}
	}
	return ""
}

func (j *Javadoc) GetCommentVersion() string {
	for _, tag := range j.Tags {
		if tag.TagName == "version" {
			return tag.Text
		}
	}
	return ""
}

func (j *Javadoc) GetCommentSince() string {
	for _, tag := range j.Tags {
		if tag.TagName == "since" {
			return tag.Text
		}
	}
	return ""
}

func (j *Javadoc) GetCommentParam() []string {
	result := []string{}
	for _, tag := range j.Tags {
		if tag.TagName == "param" {
			result = append(result, tag.Text)
		}
	}
	return result
}

func (j *Javadoc) GetCommentThrows() string {
	for _, tag := range j.Tags {
		if tag.TagName == "throws" {
			return tag.Text
		}
	}
	return ""
}

func (j *Javadoc) GetCommentReturn() string {
	for _, tag := range j.Tags {
		if tag.TagName == "return" {
			return tag.Text
		}
	}
	return ""
}

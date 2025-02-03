package java

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
)

func ParseJavadocTags(commentContent string) *model.Javadoc {
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

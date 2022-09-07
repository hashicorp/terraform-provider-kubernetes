package mdplain

import "github.com/russross/blackfriday"

// Clean runs a VERY naive cleanup of markdown text to make it more palatable as plain text.
func PlainMarkdown(md string) (string, error) {
	pt := &Text{}

	html := blackfriday.MarkdownOptions([]byte(md), pt, blackfriday.Options{})

	return string(html), nil
}

package renderer

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

var (
	unsafeMarkdown = goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.Typographer,
			extension.GFM,
			emoji.Emoji,
			highlighting.NewHighlighting(
				// similar to the HTML rendering, we don't care about the style here
				highlighting.WithCustomStyle(styles.Fallback),
				highlighting.WithFormatOptions(
					html.WithClasses(true),
					html.WithAllClasses(true),
				),
			)),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			// this is ✨dangerous✨, but we're passing the output to bluemonday
			gmhtml.WithUnsafe(),
		),
	)
)

func ToMarkdown(fileContent []byte) (template.HTML, error) {
	unsafeMarkdownHTML := bytes.NewBuffer(nil)

	if err := unsafeMarkdown.Convert(fileContent, unsafeMarkdownHTML); err != nil {
		return "", err
	}

	sanitized := htmlSanitizer.Sanitize(unsafeMarkdownHTML.String())
	wrapped := fmt.Sprintf("<div class=\"markdown\">%s</div>", sanitized)

	return template.HTML(wrapped), nil
}

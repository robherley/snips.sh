package renderer

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

var (
	md = goldmark.New(
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

	sanitizer = bluemonday.UGCPolicy().
			AllowAttrs("align").Globally().
			AllowAttrs("width").Globally()
)

func ToMarkdown(fileContent []byte) (template.HTML, error) {
	mdHTML := bytes.NewBuffer(nil)

	if err := md.Convert(fileContent, mdHTML); err != nil {
		return "", err
	}

	sanitized := sanitizer.Sanitize(mdHTML.String())
	wrapped := fmt.Sprintf("<div class=\"markdown\">%s</div>", sanitized)

	return template.HTML(wrapped), nil
}

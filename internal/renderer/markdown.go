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
)

var md = goldmark.New(
	goldmark.WithExtensions(
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
	goldmark.WithRendererOptions(),
)

func ToMarkdown(fileContent []byte) (template.HTML, error) {
	mdHTML := bytes.NewBuffer(nil)

	if err := md.Convert(fileContent, mdHTML); err != nil {
		return "", err
	}

	wrapped := fmt.Sprintf("<div class='markdown-body p-5'>%s</div>", mdHTML.String())

	return template.HTML(wrapped), nil
}

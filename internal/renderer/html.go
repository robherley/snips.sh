package renderer

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

// ToSyntaxHighlightedHTML returns HTML of the syntax highlighted code via Chroma
func ToSyntaxHighlightedHTML(fileType string, fileContent []byte) (template.HTML, error) {
	lexer := GetLexer(fileType)

	it, err := lexer.Tokenise(nil, string(fileContent))
	if err != nil {
		return "", err
	}

	formatter := html.New(
		html.WithClasses(true),
		html.WithAllClasses(true),
		html.WithLineNumbers(true),
		html.WithLinkableLineNumbers(true, "L"),
	)

	chromaHTML := bytes.NewBuffer(nil)
	// using fallback style because we'll use custom prebaked CSS
	formatter.Format(chromaHTML, styles.Fallback, it)
	if err != nil {
		return "", err
	}

	wrapped := fmt.Sprintf("<div class=\"code\">%s</div>", chromaHTML.String())
	return template.HTML(wrapped), nil
}

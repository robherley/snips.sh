package renderer

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"sync"

	"github.com/alecthomas/chroma/v2/formatters/html"
)

var (
	formatter = html.New(
		html.WithClasses(true),
		html.WithAllClasses(true),
		html.WithLineNumbers(true),
		html.WithLinkableLineNumbers(true, "L"),
	)

	syntaxCSS     template.CSS
	syntaxCSSOnce sync.Once
)

// ToSyntaxHighlightedHTML returns HTML of the syntax highlighted code via Chroma
func ToSyntaxHighlightedHTML(fileType string, fileContent []byte) (template.HTML, error) {
	lexer := GetLexer(fileType)

	it, err := lexer.Tokenise(nil, string(fileContent))
	if err != nil {
		return "", err
	}

	chromaHTML := bytes.NewBuffer(nil)
	err = formatter.Format(chromaHTML, GetStyle(), it)
	if err != nil {
		return "", err
	}

	wrapped := fmt.Sprintf("<div class=\"code\">%s</div>", chromaHTML.String())
	return template.HTML(wrapped), nil
}

func GetSyntaxCSS() template.CSS {
	syntaxCSSOnce.Do(func() {
		chromaCSS := bytes.NewBuffer(nil)
		err := formatter.WriteCSS(chromaCSS, GetStyle())
		if err != nil {
			return
		}
		syntaxCSS = template.CSS(chromaCSS.String())
	})
	return syntaxCSS
}

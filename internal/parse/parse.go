package parse

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/robherley/snips.sh/internal/db"
)

const (
	DefaultStyleName = "base16-snazzy"
	FallbackName     = "text"
)

type FileOutput struct {
	FileType string
	CSS      template.CSS
	HTML     template.HTML
}

func File(file *db.File) (*FileOutput, error) {
	var lexer chroma.Lexer
	if file.Extension != nil {
		lexer = lexers.Match(fmt.Sprintf("f.%s", *file.Extension))
	} else {
		lexer = lexers.Analyse(string(file.Content))
	}

	if lexer == nil {
		lexer = lexers.Fallback
	}

	it, err := lexer.Tokenise(nil, string(file.Content))
	if err != nil {
		return nil, err
	}

	formatter := html.New(html.WithClasses(true), html.WithLineNumbers(true), html.WithLinkableLineNumbers(true, "L"))

	// TODO(robherley): make this configurable, and cache styles
	style := styles.Get(DefaultStyleName)
	if style == nil {
		style = styles.Fallback
	}

	chromaCSS := bytes.NewBuffer(nil)
	err = formatter.WriteCSS(chromaCSS, style)
	if err != nil {
		return nil, err
	}

	chromaHTML := bytes.NewBuffer(nil)
	formatter.Format(chromaHTML, style, it)
	if err != nil {
		return nil, err
	}

	fileType := strings.ToLower(lexer.Config().Name)
	if fileType == "fallback" {
		fileType = "text"
	}

	return &FileOutput{
		FileType: fileType,
		CSS:      template.CSS(chromaCSS.String()),
		HTML:     template.HTML(chromaHTML.String()),
	}, nil
}

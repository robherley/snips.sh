package parser

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

const (
	DefaultStyleName = "base16-snazzy"
)

var (
	FallbackLexer = lexers.Plaintext
)

type LexedFile struct {
	CSS  template.CSS
	HTML template.HTML
}

// LexFile returns the CSS/HTML for a given file type and content.
func LexFile(fileType string, fileContent []byte) (*LexedFile, error) {
	lexer := GetLexer(fileType)

	it, err := lexer.Tokenise(nil, string(fileContent))
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

	return &LexedFile{
		CSS:  template.CSS(chromaCSS.String()),
		HTML: template.HTML(chromaHTML.String()),
	}, nil
}

func Analyze(content string) chroma.Lexer {
	lexer := lexers.Analyse(content)
	if lexer == nil {
		lexer = FallbackLexer
	}

	return lexer
}

// GetLexer returns the lexer for the given name, or the fallback lexer if the lexer is not found.
func GetLexer(name string) chroma.Lexer {
	lexer := lexers.Get(name)
	if lexer == nil {
		lexer = FallbackLexer
	}

	return lexer
}

// DetectFileType returns the type of the file based on the content and the hint.
// If the content's mimetype is not detected as text/plain, it returns "binary"
func DetectFileType(content []byte, hint string) string {
	detectedContentType := http.DetectContentType(content)

	if !strings.Contains(detectedContentType, "text/plain") {
		return "binary"
	}

	var lexer chroma.Lexer
	if hint != "" {
		lexer = GetLexer(hint)
	} else {
		lexer = Analyze(string(content))
	}

	return strings.ToLower(lexer.Config().Name)
}

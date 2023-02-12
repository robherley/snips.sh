package renderer

import (
	"bytes"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/styles"
)

const (
	TermFormatter = "terminal256"
	TermStyle     = "dracula"
)

// ToSyntaxHighlightedTerm returns ANSI of the syntax highlighted code via Chroma
func ToSyntaxHighlightedTerm(fileType string, fileContent []byte) (string, error) {
	if fileType == "binary" {
		return "The file is not displayed because it has been detected as binary data.", nil
	}

	lexer := GetLexer(fileType)

	it, err := lexer.Tokenise(nil, string(fileContent))
	if err != nil {
		return "", err
	}

	style := styles.Get(TermStyle)
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get(TermFormatter)

	chromaTerm := bytes.NewBuffer(nil)
	formatter.Format(chromaTerm, style, it)
	if err != nil {
		return "", err
	}

	return chromaTerm.String(), nil
}

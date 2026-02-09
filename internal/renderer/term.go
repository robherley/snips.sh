package renderer

import (
	"bytes"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/robherley/snips.sh/internal/snips"
)

const TermFormatter = "terminal256"

// ToSyntaxHighlightedTerm returns ANSI of the syntax highlighted code via Chroma
func ToSyntaxHighlightedTerm(fileType string, fileContent []byte) (string, error) {
	if fileType == snips.FileTypeBinary {
		return "The file is not displayed because it has been detected as binary data.", nil
	}

	lexer := GetLexer(fileType)

	it, err := lexer.Tokenise(nil, string(fileContent))
	if err != nil {
		return "", err
	}

	chromaTerm := bytes.NewBuffer(nil)
	err = formatters.Get(TermFormatter).Format(chromaTerm, GetStyle(), it)
	if err != nil {
		return "", err
	}

	return chromaTerm.String(), nil
}

package renderer

import (
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

var FallbackLexer chroma.Lexer = chroma.MustNewLexer(&chroma.Config{
	Name:      "plaintext",
	Filenames: []string{"*"},
	Priority:  -1,
}, lexers.PlaintextRules)

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

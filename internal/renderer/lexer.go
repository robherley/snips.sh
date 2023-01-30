package renderer

import (
	"net/http"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

var (
	FallbackLexer = lexers.Plaintext
)

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

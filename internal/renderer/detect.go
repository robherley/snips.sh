package renderer

import (
	"net/http"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/robherley/snips.sh/internal/snips"
)

const (
	// MinimumContentGuessLength is the minimum length of the content to use AI guessing, smaller content will use the fallback lexer.
	MinimumContentGuessLength = 64
)

// DetectFileType returns the type of the file based on the content and the hint.
// If useGuesser is true, it will try to guess the type of the file using AI guessing.
// If the content's mimetype is not detected as text/plain, it returns "binary"
func DetectFileType(content []byte, hint string, useGuesser bool) string {
	detectedContentType := http.DetectContentType(content)

	if !strings.Contains(detectedContentType, "text/") {
		return snips.FileTypeBinary
	}

	var lexer chroma.Lexer
	switch {
	case hint != "":
		lexer = GetLexer(hint)
	case useGuesser && len(content) >= MinimumContentGuessLength:
		if guess := Guess(string(content)); guess != "" {
			lexer = GetLexer(guess)
		}
	default:
		lexer = Analyze(string(content))
	}

	return strings.ToLower(lexer.Config().Name)
}

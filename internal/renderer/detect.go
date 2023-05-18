package renderer

import (
	"net/http"
	"strings"

	"github.com/robherley/snips.sh/internal/snips"
)

const (
	// MinimumContentGuessLength is the minimum length of the content to use guesslang, smaller content will use the fallback lexer.
	MinimumContentGuessLength = 64
)

// DetectFileType returns the type of the file based on the content and the hint.
// If useGuesser is true, it will try to guess the type of the file using guesslang.
// If the content's mimetype is not detected as text/plain, it returns "binary"
func DetectFileType(content []byte, hint string, useGuesser bool) string {
	detectedContentType := http.DetectContentType(content)

	if !strings.Contains(detectedContentType, "text/") {
		return snips.FileTypeBinary
	}

	lexer := FallbackLexer
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

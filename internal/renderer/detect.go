package renderer

import (
	"net/http"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/robherley/guesslang-go/pkg/guesser"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/rs/zerolog/log"
)

var (
	guesslang *guesser.Guesser
)

func init() {
	var err error
	guesslang, err = guesser.New()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize guesslang")
	}
}

// DetectFileType returns the type of the file based on the content and the hint.
// If useGuesser is true, it will try to guess the type of the file using guesslang.
// If the content's mimetype is not detected as text/plain, it returns "binary"
func DetectFileType(content []byte, hint string, useGuesser bool) string {
	detectedContentType := http.DetectContentType(content)

	if !strings.Contains(detectedContentType, "text/") {
		return snips.FileTypeBinary
	}

	var lexer chroma.Lexer
	if hint != "" {
		lexer = GetLexer(hint)
	} else if useGuesser {
		answer, err := guesslang.Guess(string(content))
		if err != nil {
			log.Warn().Err(err).Msg("failed to guess the file type")
		} else if answer.Reliable {
			guess := strings.ToLower(answer.Predictions[0].Language)
			lexer = GetLexer(guess)
		}
	} else {
		lexer = Analyze(string(content))
	}

	return strings.ToLower(lexer.Config().Name)
}

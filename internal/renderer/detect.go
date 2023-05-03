package renderer

import (
	"net/http"
	"strings"
	"time"

	"github.com/armon/go-metrics"
	"github.com/robherley/guesslang-go/pkg/guesser"
	"github.com/robherley/snips.sh/internal/snips"
	"github.com/rs/zerolog/log"
)

const (
	// MinimumContentGuessLength is the minimum length of the content to use guesslang, smaller content will use the fallback lexer.
	MinimumContentGuessLength = 64
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

	lexer := FallbackLexer
	switch {
	case hint != "":
		lexer = GetLexer(hint)
	case useGuesser && len(content) >= MinimumContentGuessLength:
		guessStart := time.Now()
		answer, err := guesslang.Guess(string(content))
		metrics.MeasureSince([]string{"guess", "duration"}, guessStart)
		if err != nil {
			log.Warn().Err(err).Msg("failed to guess the file type")
		} else if answer.Reliable {
			guess := strings.ToLower(answer.Predictions[0].Language)
			lexer = GetLexer(guess)
		}
	default:
		lexer = Analyze(string(content))
	}

	return strings.ToLower(lexer.Config().Name)
}

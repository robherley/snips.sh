//go:build amd64 && !noguesser

package renderer

import (
	"strings"
	"time"

	"github.com/armon/go-metrics"
	"github.com/robherley/guesslang-go/pkg/guesser"
	"github.com/rs/zerolog/log"
)

var guesslang *guesser.Guesser

func init() {
	var err error
	guesslang, err = guesser.New()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize guesslang")
	}
}

func Guess(content string) string {
	guessStart := time.Now()
	answer, err := guesslang.Guess(content)
	metrics.MeasureSince([]string{"guess", "duration"}, guessStart)
	if err != nil {
		log.Warn().Err(err).Msg("failed to guess the file type")
		return ""
	}

	return strings.ToLower(answer.Predictions[0].Language)
}

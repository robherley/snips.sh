//go:build !noguesser

package renderer

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/armon/go-metrics"
)

type magikaResult struct {
	Path   string `json:"path"`
	Result struct {
		Status string `json:"status"`
		Value  struct {
			Output struct {
				Label string `json:"label"`
			} `json:"output"`
		} `json:"value"`
	} `json:"result"`
}

func Guess(content string) string {
	guessStart := time.Now()
	defer metrics.MeasureSince([]string{"guess", "duration"}, guessStart)

	// Call magika CLI with stdin input and JSON output
	cmd := exec.Command("magika", "-", "--json")
	cmd.Stdin = strings.NewReader(content)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		slog.Warn("failed to run magika", "err", err, "stderr", stderr.String())
		return ""
	}

	// Parse JSON output
	var results []magikaResult
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		slog.Warn("failed to parse magika output", "err", err, "output", stdout.String())
		return ""
	}

	if len(results) == 0 || results[0].Result.Status != "ok" {
		slog.Warn("magika returned no results or error status")
		return ""
	}

	label := strings.ToLower(results[0].Result.Value.Output.Label)

	// Map magika labels to chroma lexer names for better compatibility
	switch label {
	case "c++":
		return "cpp"
	case "c#":
		return "csharp"
	case "objective-c":
		return "objectivec"
	case "shell":
		return "bash"
	default:
		return label
	}
}
